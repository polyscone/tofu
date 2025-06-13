package cache

import (
	"sync"
	"time"
)

type Item[V any] struct {
	value     V
	expiresAt time.Time
}

type Cache[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]Item[V]
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{data: make(map[K]Item[V])}
}

func (c *Cache[K, V]) Load(key K) (zero V, _ bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.data[key]
	if ok && (item.expiresAt.IsZero() || time.Now().Before(item.expiresAt)) {
		return item.value, true
	}

	return zero, false
}

func (c *Cache[K, V]) Store(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.data[key] = Item[V]{
		value:     value,
		expiresAt: expiresAt,
	}
}

func (c *Cache[K, V]) LoadOrMaybeStore(key K, value func() (V, time.Duration, error)) (V, error) {
	c.mu.RLock()

	if item, ok := c.data[key]; ok && (item.expiresAt.IsZero() || time.Now().Before(item.expiresAt)) {
		c.mu.RUnlock()

		return item.value, nil
	}

	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.data[key]; ok && (item.expiresAt.IsZero() || time.Now().Before(item.expiresAt)) {
		return item.value, nil
	}

	v, ttl, err := value()
	if err == nil {
		var expiresAt time.Time
		if ttl > 0 {
			expiresAt = time.Now().Add(ttl)
		}

		c.data[key] = Item[V]{
			value:     v,
			expiresAt: expiresAt,
		}
	}

	return v, err
}

func (c *Cache[K, V]) LoadOrStore(key K, value func() (V, time.Duration)) V {
	v, _ := c.LoadOrMaybeStore(key, func() (V, time.Duration, error) {
		v, ttl := value()

		return v, ttl, nil
	})

	return v
}

func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.data)
}

func (c *Cache[K, V]) ClearExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.data, key)
		}
	}
}
