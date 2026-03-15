package middleware

import (
	"container/list"
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CacheConfig configures the caching middleware.
type CacheConfig struct {
	// MaxEntries is the maximum number of responses held in memory.
	// When full, the least recently used entry is evicted. Default: 256.
	MaxEntries int

	// DefaultTTL is the TTL applied when the response does not carry a
	// Cache-Control max-age directive. Default: 5 minutes.
	DefaultTTL time.Duration

	// CacheableMethods lists the HTTP methods whose responses may be cached.
	// Default: {"GET"}.
	CacheableMethods []string

	// CacheableStatuses lists the response status codes that may be cached.
	// Default: {200}.
	CacheableStatuses []int
}

// DefaultCacheConfig returns a CacheConfig with sensible defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxEntries:        256,
		DefaultTTL:        5 * time.Minute,
		CacheableMethods:  []string{"GET"},
		CacheableStatuses: []int{200},
	}
}

// cacheEntry is a single entry stored in the LRU cache.
type cacheEntry struct {
	key    string
	resp   *Response
	expiry time.Time     // zero means no expiry
	elem   *list.Element // back-pointer into the LRU list for O(1) moves
}

// Cache is a middleware that stores and serves responses from an in-memory
// LRU cache with TTL eviction.
//
// On a cache hit the stored response is returned without calling next.
// On a cache miss (or expired entry) next is called; if the response is
// cacheable (method + status allowed, no "no-store" directive) it is stored.
//
// Cache-Control support:
//   - "no-store": response is never stored.
//   - "max-age=N": entry TTL is set to N seconds, overriding DefaultTTL.
type Cache struct {
	cfg      CacheConfig
	mu       sync.Mutex
	lru      *list.List               // front = most recently used
	entries  map[string]*list.Element // cacheKey → LRU list element
	methods  map[string]bool
	statuses map[int]bool
}

// NewCache creates a caching middleware from cfg.
func NewCache(cfg CacheConfig) *Cache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 256
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	if len(cfg.CacheableMethods) == 0 {
		cfg.CacheableMethods = []string{"GET"}
	}
	if len(cfg.CacheableStatuses) == 0 {
		cfg.CacheableStatuses = []int{200}
	}

	methods := make(map[string]bool, len(cfg.CacheableMethods))
	for _, m := range cfg.CacheableMethods {
		methods[strings.ToUpper(m)] = true
	}
	statuses := make(map[int]bool, len(cfg.CacheableStatuses))
	for _, s := range cfg.CacheableStatuses {
		statuses[s] = true
	}

	return &Cache{
		cfg:      cfg,
		lru:      list.New(),
		entries:  make(map[string]*list.Element),
		methods:  methods,
		statuses: statuses,
	}
}

// ProcessRequest implements Middleware. It returns a cached response on hit,
// or calls next on miss and conditionally stores the result.
func (c *Cache) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}

	// Methods not in the allow-list bypass the cache entirely.
	if !c.methods[method] {
		return next(ctx, req)
	}

	key := method + " " + req.URL

	if cached := c.get(key); cached != nil {
		return cached, nil
	}

	resp, err := next(ctx, req)
	if err != nil || resp == nil {
		return resp, err
	}

	if c.statuses[resp.StatusCode] && !hasNoStore(resp) {
		ttl := responseTTL(resp, c.cfg.DefaultTTL)
		c.set(key, resp, ttl)
	}

	return resp, nil
}

// get returns the cached response for key, or nil on miss or expiry.
func (c *Cache) get(key string) *Response {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		return nil
	}
	entry := elem.Value.(*cacheEntry)

	if !entry.expiry.IsZero() && time.Now().After(entry.expiry) {
		c.evict(elem)
		return nil
	}

	c.lru.MoveToFront(elem)
	return entry.resp
}

// set stores resp under key with the given TTL, evicting LRU entries as needed.
func (c *Cache) set(key string, resp *Response, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update in place if already present.
	if elem, ok := c.entries[key]; ok {
		c.lru.MoveToFront(elem)
		e := elem.Value.(*cacheEntry)
		e.resp = resp
		e.expiry = expiryTime(ttl)
		return
	}

	// Evict LRU entries until we are under capacity.
	for len(c.entries) >= c.cfg.MaxEntries {
		c.evict(c.lru.Back())
	}

	entry := &cacheEntry{
		key:    key,
		resp:   resp,
		expiry: expiryTime(ttl),
	}
	elem := c.lru.PushFront(entry)
	entry.elem = elem
	c.entries[key] = elem
}

// evict removes elem from the LRU list and the entry map.
// Must be called with c.mu held.
func (c *Cache) evict(elem *list.Element) {
	if elem == nil {
		return
	}
	c.lru.Remove(elem)
	delete(c.entries, elem.Value.(*cacheEntry).key)
}

// Len returns the current number of entries held in the cache.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// hasNoStore reports whether resp carries a "Cache-Control: no-store" directive.
func hasNoStore(resp *Response) bool {
	cc := resp.Headers["Cache-Control"]
	if cc == "" {
		return false
	}
	for _, d := range strings.Split(cc, ",") {
		if strings.TrimSpace(strings.ToLower(d)) == "no-store" {
			return true
		}
	}
	return false
}

// responseTTL returns the TTL for a cached response.
// It prefers the "max-age=N" value from Cache-Control; falls back to defaultTTL.
func responseTTL(resp *Response, defaultTTL time.Duration) time.Duration {
	if cc := resp.Headers["Cache-Control"]; cc != "" {
		for _, d := range strings.Split(cc, ",") {
			d = strings.TrimSpace(strings.ToLower(d))
			if strings.HasPrefix(d, "max-age=") {
				val := strings.TrimPrefix(d, "max-age=")
				if n, err := strconv.Atoi(val); err == nil && n > 0 {
					return time.Duration(n) * time.Second
				}
			}
		}
	}
	return defaultTTL
}

// expiryTime returns time.Now().Add(ttl), or a zero time.Time for no expiry.
func expiryTime(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}
