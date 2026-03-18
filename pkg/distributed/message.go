// Package distributed provides Kafka-based distributed crawling primitives
// including URL producers, worker consumers, job management, and failure
// recovery with dead letter queues.
package distributed

import (
	"encoding/json"
	"time"
)

// CrawlTask represents a unit of work sent through Kafka for distributed
// crawling. Tasks are partitioned by Domain to ensure crawl politeness
// (requests to the same domain are processed by the same consumer).
type CrawlTask struct {
	// ID uniquely identifies this task.
	ID string `json:"id"`

	// JobID identifies the crawl job this task belongs to.
	JobID string `json:"job_id"`

	// URL is the target URL to crawl.
	URL string `json:"url"`

	// Domain is extracted from URL for Kafka partitioning.
	Domain string `json:"domain"`

	// Depth is the link depth from the seed URL.
	Depth int `json:"depth"`

	// MaxDepth is the maximum crawl depth allowed.
	MaxDepth int `json:"max_depth"`

	// Priority controls processing order (higher = sooner).
	Priority int `json:"priority"`

	// Headers are additional HTTP headers to send with the request.
	Headers map[string]string `json:"headers,omitempty"`

	// Meta carries arbitrary key-value metadata through the pipeline.
	Meta map[string]string `json:"meta,omitempty"`

	// RetryCount tracks how many times this task has been retried.
	RetryCount int `json:"retry_count"`

	// CreatedAt is when the task was first created.
	CreatedAt time.Time `json:"created_at"`
}

// Encode serializes the task to JSON bytes for Kafka message values.
func (t *CrawlTask) Encode() ([]byte, error) {
	return json.Marshal(t)
}

// DecodeTask deserializes a CrawlTask from JSON bytes.
func DecodeTask(data []byte) (*CrawlTask, error) {
	var task CrawlTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// ControlMessage is sent on a dedicated control topic to coordinate
// distributed workers (pause, resume, stop jobs).
type ControlMessage struct {
	// Type is the control command type.
	Type ControlType `json:"type"`

	// JobID identifies which job this control message targets.
	JobID string `json:"job_id"`

	// Timestamp is when the control message was issued.
	Timestamp time.Time `json:"timestamp"`
}

// ControlType enumerates the control commands.
type ControlType string

const (
	ControlPause  ControlType = "pause"
	ControlResume ControlType = "resume"
	ControlStop   ControlType = "stop"
)

// Encode serializes the control message to JSON bytes.
func (c *ControlMessage) Encode() ([]byte, error) {
	return json.Marshal(c)
}

// DecodeControl deserializes a ControlMessage from JSON bytes.
func DecodeControl(data []byte) (*ControlMessage, error) {
	var msg ControlMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
