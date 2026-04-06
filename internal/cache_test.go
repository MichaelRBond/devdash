package internal

import (
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	c := NewCache[string](1 * time.Minute)
	c.Set([]string{"hello", "world"})

	items, ok := c.Get()
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(items) != 2 || items[0] != "hello" {
		t.Errorf("unexpected items: %v", items)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := NewCache[string](1 * time.Millisecond)
	c.Set([]string{"hello"})

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get()
	if ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestCacheEmpty(t *testing.T) {
	c := NewCache[string](1 * time.Minute)

	_, ok := c.Get()
	if ok {
		t.Error("expected cache miss on empty cache")
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := NewCache[string](1 * time.Minute)
	c.Set([]string{"hello"})
	c.Invalidate()

	_, ok := c.Get()
	if ok {
		t.Error("expected cache miss after invalidation")
	}
}

func TestCacheMinRefreshInterval(t *testing.T) {
	c := NewCache[string](1 * time.Minute)
	c.Set([]string{"hello"})

	if c.ShouldThrottle(10 * time.Second) {
		t.Error("should not throttle on first check")
	}

	// Simulate immediate re-fetch attempt
	c.MarkFetchStart()
	if !c.ShouldThrottle(10 * time.Second) {
		t.Error("should throttle within min interval")
	}
}
