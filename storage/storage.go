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

func (c *ConcurrentSet[T]) Add(values ...T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, v := range values {
		c.set[v] = struct{}{}
	}
}

func (c *ConcurrentSet[T]) Present(value T) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, prs := c.set[value]
	return prs
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

func (c *ConcurrentSet[T]) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.set)
}

func (c *ConcurrentSet[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.set = make(map[T]struct{})
}

func (c *ConcurrentSet[T]) Delete(val T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.set, val)
}
