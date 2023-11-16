package safe_channel

import (
	"errors"
	"sync"
)

type SafeChannel[T any] struct {
	sync.RWMutex

	closed bool
	Ch     chan T
}

func NewSafeChannel[T any](cap int) *SafeChannel[T] {
	return &SafeChannel[T]{
		Ch: make(chan T, cap),
	}
}

func (self *SafeChannel[T]) Write(t T) error {
	self.RLock()
	defer self.RUnlock()
	if self.closed {
		return errors.New("queue is closed")
	}
	self.Ch <- t
	return nil
}

func (self *SafeChannel[T]) Close() {
	self.Lock()
	defer self.Unlock()
	if self.closed {
		return
	}
	close(self.Ch)
	self.closed = true
}
