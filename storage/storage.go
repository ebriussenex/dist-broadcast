package storage

import (
	"sync"
)

type ConcurrentSet[T comparable] struct {
	mu  *sync.RWMutex
	set map[T]struct{}
}

func Init[T comparable]() *ConcurrentSet[T] {
	return &ConcurrentSet[T]{
		mu: &sync.RWMutex{}, set: make(map[T]struct{}),
	}
}

func (c *ConcurrentSet[T]) Add(value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.set[value] = struct{}{}
}

func (c *ConcurrentSet[T]) GetAll() []T {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]T, 0, len(c.set))
	for k := range c.set {
		res = append(res, k)
	}

	return res
}
