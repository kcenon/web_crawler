package client

import (
	"net/http"
	"sync/atomic"
)

// PoolStats holds connection pool statistics.
type PoolStats struct {
	TotalRequests int64
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
