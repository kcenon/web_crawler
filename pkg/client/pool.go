package client

import (
	"bytes"
	"net/http"
	"sync"
	"sync/atomic"
)

// PoolStats holds connection pool statistics.
type PoolStats struct {
	TotalRequests int64
}

// bufPool is a shared pool of *bytes.Buffer used to read response bodies
// without allocating a new buffer per request.
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// acquireBuf returns a reset buffer from the pool.
func acquireBuf() *bytes.Buffer {
	buf := bufPool.Get().(*bytes.Buffer) //nolint:errcheck // type is always *bytes.Buffer
	buf.Reset()
	return buf
}

// releaseBuf returns a buffer to the pool.
func releaseBuf(buf *bytes.Buffer) {
	// Avoid retaining very large buffers that would pin memory.
	const maxRetainBytes = 1 << 20 // 1 MiB
	if buf.Cap() <= maxRetainBytes {
		bufPool.Put(buf)
	}
}

// Pool manages HTTP transport connections and tracks pool statistics.
type Pool struct {
	transport     *http.Transport
	totalRequests atomic.Int64
}

// newPool creates a new Pool from the given TransportConfig.
func newPool(cfg TransportConfig) (*Pool, error) {
	t, err := buildTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &Pool{transport: t}, nil
}

// stats returns current pool statistics.
func (p *Pool) stats() PoolStats {
	return PoolStats{
		TotalRequests: p.totalRequests.Load(),
	}
}

func (p *Pool) recordRequest() {
	p.totalRequests.Add(1)
}

func (p *Pool) close() {
	p.transport.CloseIdleConnections()
}
