package distributed

import (
	"sync/atomic"
	"time"
)

// Stats tracks distributed crawling metrics with atomic operations for
// thread-safe access from multiple workers.
type Stats struct {
	processed  atomic.Int64
	succeeded  atomic.Int64
	failed     atomic.Int64
	dlqCount   atomic.Int64
	totalNanos atomic.Int64
}

// StatsSnapshot is a point-in-time copy of the stats counters.
type StatsSnapshot struct {
	Processed   int64         `json:"processed"`
	Succeeded   int64         `json:"succeeded"`
	Failed      int64         `json:"failed"`
	DLQCount    int64         `json:"dlq_count"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// NewStats creates a new Stats tracker.
func NewStats() *Stats {
	return &Stats{}
}

// RecordProcessed increments the total tasks processed counter.
func (s *Stats) RecordProcessed() {
	s.processed.Add(1)
}

// RecordSuccess increments the successful tasks counter.
func (s *Stats) RecordSuccess() {
	s.succeeded.Add(1)
}

// RecordError increments the failed tasks counter.
func (s *Stats) RecordError() {
	s.failed.Add(1)
}

// RecordDLQ increments the dead letter queue counter.
func (s *Stats) RecordDLQ() {
	s.dlqCount.Add(1)
}

// RecordDuration adds a task processing duration to the total.
func (s *Stats) RecordDuration(d time.Duration) {
	s.totalNanos.Add(int64(d))
}

// Snapshot returns a point-in-time copy of all counters.
func (s *Stats) Snapshot() StatsSnapshot {
	processed := s.processed.Load()
	var avg time.Duration
	if processed > 0 {
		avg = time.Duration(s.totalNanos.Load() / processed)
	}

	return StatsSnapshot{
		Processed:   processed,
		Succeeded:   s.succeeded.Load(),
		Failed:      s.failed.Load(),
		DLQCount:    s.dlqCount.Load(),
		AvgDuration: avg,
	}
}
