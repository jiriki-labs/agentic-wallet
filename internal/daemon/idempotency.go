package daemon

import (
	"sync"
	"time"
)

type cachedResponse struct {
	status  int
	body    []byte
	expires time.Time
}

// idempotencyCache is a simple in-memory LRU-style TTL cache for idempotency keys.
// For MVP: fixed-capacity map with expiry; production would use a proper LRU.
type idempotencyCache struct {
	mu      sync.Mutex
	entries map[string]*cachedResponse
	cap     int
	ttl     time.Duration
	stopCh  chan struct{}
}

func newIdempotencyCache(capacity int, ttl time.Duration) *idempotencyCache {
	c := &idempotencyCache{
		entries: make(map[string]*cachedResponse, capacity),
		cap:     capacity,
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}
	go c.evictLoop()
	return c
}

func (c *idempotencyCache) get(key string) (*cachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(r.expires) {
		delete(c.entries, key)
		return nil, false
	}
	return r, true
}

func (c *idempotencyCache) set(key string, status int, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Evict an expired entry first; if still at capacity, drop the oldest by expiry.
	if len(c.entries) >= c.cap {
		now := time.Now()
		evicted := false
		for k, v := range c.entries {
			if now.After(v.expires) {
				delete(c.entries, k)
				evicted = true
				break
			}
		}
		if !evicted {
			var oldestKey string
			var oldestExp time.Time
			first := true
			for k, v := range c.entries {
				if first || v.expires.Before(oldestExp) {
					oldestKey = k
					oldestExp = v.expires
					first = false
				}
			}
			if oldestKey != "" {
				delete(c.entries, oldestKey)
			}
		}
	}
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	c.entries[key] = &cachedResponse{
		status:  status,
		body:    bodyCopy,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *idempotencyCache) evictLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, v := range c.entries {
				if now.After(v.expires) {
					delete(c.entries, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

func (c *idempotencyCache) stop() {
	close(c.stopCh)
}
