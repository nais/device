package kolide

import (
	"encoding/json"
	"sync"
)

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
	if c.cache == nil {
		c.cache = make(map[K]V)
	}
	c.cache[key] = value
}

func (c *Cache[K, V]) Replace(cache map[K]V) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = cache
}

func (c *Cache[K, V]) MarshalJSON() ([]byte, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return json.Marshal(c.cache)
}

func (c *Cache[K, V]) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.cache)
}
