package crawler

import (
	"context"
	"fmt"
	"sync"
)

// service is a concrete implementation of the Service interface.
// It manages multiple named crawler instances.
type service struct {
	mu        sync.RWMutex
	instances map[string]*managedInstance
}

type managedInstance struct {
	engine *Engine
	cancel context.CancelFunc
}

// NewService creates a new Service that manages crawler instances.
func NewService() Service {
	return &service{
		instances: make(map[string]*managedInstance),
	}
}

// Crawl performs a one-shot crawl of the given URLs and returns results.
func (s *service) Crawl(ctx context.Context, urls []string, cfg *CrawlConfig) ([]*Result, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs provided")
	}

	engineCfg := Config{
		MaxDepth: int(cfg.MaxDepth),
		MaxPages: int(cfg.MaxPages),
	}

	e, err := newEngine(engineCfg)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}

	var (
		mu      sync.Mutex
		results []*Result
	)

	e.OnResponse(func(resp *CrawlResponse) {
		r := &Result{
			URL:        resp.Request.URL,
			Content:    string(resp.Body),
			StatusCode: resp.StatusCode,
		}
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	})

	e.OnError(func(req *CrawlRequest, err error) {
		r := &Result{
			URL:   req.URL,
			Error: err,
		}
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	})

	if err := e.AddURLs(urls); err != nil {
		return nil, fmt.Errorf("add URLs: %w", err)
	}

	if err := e.Start(ctx); err != nil {
		return nil, fmt.Errorf("start crawl: %w", err)
	}

	_ = e.Wait()

	return results, nil
}

// Start creates and starts a named crawler instance.
func (s *service) Start(ctx context.Context, id string, cfg *CrawlConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[id]; exists {
		return fmt.Errorf("crawler %q already running", id)
	}

	engineCfg := Config{
		MaxDepth: int(cfg.MaxDepth),
		MaxPages: int(cfg.MaxPages),
	}

	e, err := newEngine(engineCfg)
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}

	crawlCtx, cancel := context.WithCancel(ctx)

	if len(cfg.URLs) > 0 {
		if err := e.AddURLs(cfg.URLs); err != nil {
			cancel()
			return fmt.Errorf("add URLs: %w", err)
		}
	}

	if err := e.Start(crawlCtx); err != nil {
		cancel()
		return fmt.Errorf("start crawler: %w", err)
	}

	s.instances[id] = &managedInstance{
		engine: e,
		cancel: cancel,
	}

	return nil
}

// Stop stops a running crawler instance and returns its final stats.
func (s *service) Stop(ctx context.Context, id string) (*Stats, error) {
	s.mu.Lock()
	inst, ok := s.instances[id]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("crawler %q not found", id)
	}
	delete(s.instances, id)
	s.mu.Unlock()

	_ = inst.engine.Stop(ctx)
	inst.cancel()

	es := inst.engine.Stats()
	return &Stats{
		PagesCrawled: es.SuccessCount,
		PagesFailed:  es.ErrorCount,
		PagesQueued:  es.RequestCount - es.SuccessCount - es.ErrorCount,
	}, nil
}

// Stats returns statistics for a running crawler instance.
func (s *service) Stats(_ context.Context, id string) (*Stats, error) {
	s.mu.RLock()
	inst, ok := s.instances[id]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("crawler %q not found", id)
	}

	es := inst.engine.Stats()
	return &Stats{
		PagesCrawled: es.SuccessCount,
		PagesFailed:  es.ErrorCount,
		PagesQueued:  es.RequestCount - es.SuccessCount - es.ErrorCount,
	}, nil
}

// AddURLs adds URLs to a running crawler instance.
func (s *service) AddURLs(_ context.Context, id string, urls []string) (int, error) {
	s.mu.RLock()
	inst, ok := s.instances[id]
	s.mu.RUnlock()

	if !ok {
		return 0, fmt.Errorf("crawler %q not found", id)
	}

	added := 0
	for _, u := range urls {
		if err := inst.engine.AddURL(u); err != nil {
			break
		}
		added++
	}
	return added, nil
}
