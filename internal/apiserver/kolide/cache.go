package kolide

import "sync"

type Cache[K comparable, V any] struct {
	mutex sync.RWMutex
	cache map[K]V
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, ok := c.cache[key]
	return value, ok
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[key] = value
}

func (c *Cache[K, V]) Replace(cache map[K]V) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = cache
}
