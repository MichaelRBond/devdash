package internal

import (
	"sync"
	"time"
)

// Cache is a generic in-memory cache with TTL.
type Cache[T any] struct {
	mu          sync.RWMutex
	items       []T
	hasData     bool
	storedAt    time.Time
	ttl         time.Duration
	lastFetchAt time.Time
}

// NewCache creates a cache with the given TTL.
func NewCache[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{ttl: ttl}
}

// Get returns cached items if they exist and haven't expired.
func (c *Cache[T]) Get() ([]T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.hasData {
		return nil, false
	}
	if time.Since(c.storedAt) > c.ttl {
		return nil, false
	}
	return c.items, true
}

// Set stores items in the cache.
func (c *Cache[T]) Set(items []T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = items
	c.hasData = true
	c.storedAt = time.Now()
}

// Invalidate clears the cache.
func (c *Cache[T]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hasData = false
	c.items = nil
}

// MarkFetchStart records the start of a fetch, for throttle checking.
func (c *Cache[T]) MarkFetchStart() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastFetchAt = time.Now()
}

// ShouldThrottle returns true if the last fetch was within minInterval.
func (c *Cache[T]) ShouldThrottle(minInterval time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastFetchAt.IsZero() {
		return false
	}
	return time.Since(c.lastFetchAt) < minInterval
}
