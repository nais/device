package triggers

import (
	"fmt"
	"sync"
)

func New[T comparable]() *StreamTriggers[T] {
	return &StreamTriggers[T]{triggers: make(map[T]chan struct{})}
}

type StreamTriggers[T comparable] struct {
	triggers map[T]chan struct{}
	lock     sync.RWMutex
}

func (t *StreamTriggers[T]) Add(id T) (<-chan struct{}, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, exists := t.triggers[id]; exists {
		return nil, fmt.Errorf("trigger already exists for %v", id)
	}

	c := make(chan struct{})
	t.triggers[id] = c

	return c, nil
}

func (t *StreamTriggers[T]) Remove(id T) {
	t.lock.Lock()
	defer t.lock.Unlock()

	c, exists := t.triggers[id]
	if !exists {
		return
	}

	close(c)
	delete(t.triggers, id)
}

func (t *StreamTriggers[T]) Trigger(id T) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	c, exists := t.triggers[id]
	if !exists {
		return
	}

	select {
	case c <- struct{}{}:
	default:
	}
}

func (t *StreamTriggers[T]) Exists(id T) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, exists := t.triggers[id]
	return exists
}

func (t *StreamTriggers[T]) Close() {
	t.lock.Lock()
	defer t.lock.Unlock()

	for id, c := range t.triggers {
		close(c)
		delete(t.triggers, id)
	}
}

func (t *StreamTriggers[T]) Length() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.triggers)
}

func (t *StreamTriggers[T]) TriggerAll() {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, c := range t.triggers {
		select {
		case c <- struct{}{}:
		default:
		}
	}
}

func (t *StreamTriggers[T]) Keys() []T {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var keys []T
	for key := range t.triggers {
		keys = append(keys, key)
	}
	return keys
}
