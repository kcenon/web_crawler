package distributed

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// ProducerConfig configures the Kafka URL producer.
type ProducerConfig struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string

	// Topic is the Kafka topic for crawl tasks.
	Topic string

	// BatchSize is the maximum number of messages per batch. Default: 100.
	BatchSize int

	// BatchTimeout is the maximum time to wait before flushing a batch.
	// Default: 1 second.
	BatchTimeout time.Duration
}

func (c *ProducerConfig) defaults() {
	if c.Topic == "" {
		c.Topic = "crawl-tasks"
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.BatchTimeout <= 0 {
		c.BatchTimeout = time.Second
	}
}

// Producer publishes CrawlTask messages to Kafka for distributed processing.
// Messages are keyed by domain to ensure all URLs from the same domain are
// routed to the same partition, enabling per-domain crawl politeness.
type Producer struct {
	writer *kafka.Writer
	config ProducerConfig
}

// NewProducer creates a Kafka producer for crawl task distribution.
func NewProducer(cfg ProducerConfig) *Producer {
	cfg.defaults()

	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Async:        false,
	}

	return &Producer{
		writer: w,
		config: cfg,
	}
}

// Publish sends a single crawl task to Kafka. The task is keyed by domain
// for partition affinity.
func (p *Producer) Publish(ctx context.Context, task *CrawlTask) error {
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if task.Domain == "" {
		task.Domain = extractDomain(task.URL)
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	data, err := task.Encode()
	if err != nil {
		return fmt.Errorf("distributed: encode task: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(task.Domain),
		Value: data,
	})
}

// PublishBatch sends multiple crawl tasks to Kafka in a single batch.
func (p *Producer) PublishBatch(ctx context.Context, tasks []*CrawlTask) error {
	msgs := make([]kafka.Message, 0, len(tasks))
	for _, task := range tasks {
		if task.ID == "" {
			task.ID = uuid.NewString()
		}
		if task.Domain == "" {
			task.Domain = extractDomain(task.URL)
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = time.Now()
		}

		data, err := task.Encode()
		if err != nil {
			return fmt.Errorf("distributed: encode task %s: %w", task.URL, err)
		}

		msgs = append(msgs, kafka.Message{
			Key:   []byte(task.Domain),
			Value: data,
		})
	}

	return p.writer.WriteMessages(ctx, msgs...)
}

// Close flushes pending messages and closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// extractDomain returns the hostname from a URL, or the raw string on failure.
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}
