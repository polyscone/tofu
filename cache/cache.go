package cache

import "sync"

type Cache[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{data: make(map[K]V)}
}

func (c *Cache[K, V]) LoadOrMaybeStore(key K, value func() (V, error)) (V, error) {
	c.mu.RLock()

	if v, ok := c.data[key]; ok {
		c.mu.RUnlock()

		return v, nil
	}

	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.data[key]; ok {
		return v, nil
	}

	v, err := value()
	if err == nil {
		c.data[key] = v
	}

	return v, err
}

func (c *Cache[K, V]) LoadOrStore(key K, value func() V) V {
	v, _ := c.LoadOrMaybeStore(key, func() (V, error) {
		return value(), nil
	})

	return v
}
