package distributed

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// TaskHandler processes a single crawl task. Returning a non-nil error
// triggers the retry/DLQ logic.
type TaskHandler func(ctx context.Context, task *CrawlTask) error

// ConsumerConfig configures the Kafka crawl task consumer.
type ConsumerConfig struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string

	// Topic is the Kafka topic for crawl tasks.
	Topic string

	// GroupID is the Kafka consumer group ID. Workers in the same group
	// share partitions for load balancing.
	GroupID string

	// Workers is the number of concurrent task processors. Default: 4.
	Workers int

	// MaxRetries is the maximum number of retries for a failed task.
	// Default: 3.
	MaxRetries int

	// RetryBackoff is the base duration for exponential retry backoff.
	// Default: 5 seconds.
	RetryBackoff time.Duration

	// DLQTopic is the dead letter queue topic for permanently failed tasks.
	// Default: "<topic>.dlq".
	DLQTopic string

	// Logger for consumer operations. If nil, slog.Default() is used.
	Logger *slog.Logger
}

func (c *ConsumerConfig) defaults() {
	if c.Topic == "" {
		c.Topic = "crawl-tasks"
	}
	if c.GroupID == "" {
		c.GroupID = "crawler-workers"
	}
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = 5 * time.Second
	}
	if c.DLQTopic == "" {
		c.DLQTopic = c.Topic + ".dlq"
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}

// Consumer reads crawl tasks from Kafka, processes them with the registered
// handler, and manages retry/DLQ for failed tasks.
type Consumer struct {
	reader  *kafka.Reader
	dlq     *kafka.Writer
	handler TaskHandler
	config  ConsumerConfig
	stats   *Stats
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewConsumer creates a Kafka consumer for distributed crawl task processing.
func NewConsumer(cfg ConsumerConfig, handler TaskHandler) *Consumer {
	cfg.defaults()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10 << 20, // 10 MB
		CommitInterval: time.Second,
	})

	dlqWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.DLQTopic,
		Balancer: &kafka.Hash{},
	}

	return &Consumer{
		reader:  reader,
		dlq:     dlqWriter,
		handler: handler,
		config:  cfg,
		stats:   NewStats(),
	}
}

// Start begins consuming tasks from Kafka with the configured number of workers.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (c *Consumer) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)

	taskCh := make(chan *CrawlTask, c.config.Workers*2)

	// Start worker goroutines.
	for i := range c.config.Workers {
		c.wg.Add(1)
		go c.worker(ctx, i, taskCh)
	}

	// Read loop.
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer close(taskCh)

		for {
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.config.Logger.Error("fetch message failed", "error", err)
				continue
			}

			task, err := DecodeTask(msg.Value)
			if err != nil {
				c.config.Logger.Error("decode task failed",
					"error", err,
					"offset", msg.Offset,
				)
				// Commit to skip malformed messages.
				_ = c.reader.CommitMessages(ctx, msg)
				c.stats.RecordError()
				continue
			}

			select {
			case taskCh <- task:
				_ = c.reader.CommitMessages(ctx, msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	c.wg.Wait()
	return ctx.Err()
}

// worker processes tasks from the channel.
func (c *Consumer) worker(ctx context.Context, id int, tasks <-chan *CrawlTask) {
	defer c.wg.Done()

	logger := c.config.Logger.With("worker", id)

	for task := range tasks {
		if ctx.Err() != nil {
			return
		}

		logger.Debug("processing task",
			"url", task.URL,
			"job_id", task.JobID,
			"retry", task.RetryCount,
		)

		c.stats.RecordProcessed()
		start := time.Now()

		err := c.handler(ctx, task)

		duration := time.Since(start)
		c.stats.RecordDuration(duration)

		if err != nil {
			c.handleFailure(ctx, task, err, logger)
		} else {
			c.stats.RecordSuccess()
		}
	}
}

// handleFailure decides whether to retry or route to DLQ.
func (c *Consumer) handleFailure(ctx context.Context, task *CrawlTask, err error, logger *slog.Logger) {
	c.stats.RecordError()

	if task.RetryCount < c.config.MaxRetries {
		task.RetryCount++
		backoff := c.config.RetryBackoff * time.Duration(1<<(task.RetryCount-1))

		logger.Warn("task failed, scheduling retry",
			"url", task.URL,
			"retry", task.RetryCount,
			"backoff", backoff,
			"error", err,
		)

		// Wait for backoff, then re-publish to the task topic.
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}

		data, encErr := task.Encode()
		if encErr != nil {
			logger.Error("encode retry task failed", "error", encErr)
			return
		}

		retryWriter := &kafka.Writer{
			Addr:  kafka.TCP(c.config.Brokers...),
			Topic: c.config.Topic,
		}
		writeErr := retryWriter.WriteMessages(ctx, kafka.Message{
			Key:   []byte(task.Domain),
			Value: data,
		})
		_ = retryWriter.Close()

		if writeErr != nil {
			logger.Error("retry publish failed", "error", writeErr)
		}
	} else {
		logger.Error("task permanently failed, routing to DLQ",
			"url", task.URL,
			"retries", task.RetryCount,
			"error", err,
		)

		c.stats.RecordDLQ()

		data, encErr := task.Encode()
		if encErr != nil {
			logger.Error("encode DLQ task failed", "error", encErr)
			return
		}

		dlqErr := c.dlq.WriteMessages(ctx, kafka.Message{
			Key:   []byte(task.Domain),
			Value: data,
		})
		if dlqErr != nil {
			logger.Error("DLQ publish failed", "error", dlqErr)
		}
	}
}

// Stop gracefully shuts down the consumer, waiting for in-flight tasks.
func (c *Consumer) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()

	var errs []error
	if err := c.reader.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close reader: %w", err))
	}
	if err := c.dlq.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close DLQ writer: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("consumer stop: %v", errs)
	}
	return nil
}

// Stats returns the consumer's runtime statistics.
func (c *Consumer) Stats() StatsSnapshot {
	return c.stats.Snapshot()
}
