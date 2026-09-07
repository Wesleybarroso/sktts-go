package services

import "sync"

type planSemaphore struct {
	ch chan struct{}
}

type ConcurrencyService struct {
	mu         sync.Mutex
	semaphores map[string]*planSemaphore
}

func NewConcurrencyService() *ConcurrencyService {
	return &ConcurrencyService{
		semaphores: make(map[string]*planSemaphore),
	}
}

func (c *ConcurrencyService) Acquire(plan string, limit int) func() {
	if limit < 1 {
		limit = 1
	}

	c.mu.Lock()

	sem, exists := c.semaphores[plan]

	if !exists || cap(sem.ch) != limit {
		sem = &planSemaphore{
			ch: make(chan struct{}, limit),
		}
		c.semaphores[plan] = sem
	}

	ch := sem.ch

	c.mu.Unlock()

	ch <- struct{}{}

	return func() {
		<-ch
	}
}
