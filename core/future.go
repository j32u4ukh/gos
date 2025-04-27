package core

import (
	"errors"
	"time"
)

type Future[T any] struct {
	value T
	err   error
	done  chan struct{}
}

func NewFuture[T any]() *Future[T] {
	return &Future[T]{
		done: make(chan struct{}),
	}
}

func (f *Future[T]) Set(value T, err error) {
	f.value = value
	f.err = err
	close(f.done)
}

func (f *Future[T]) Get() (T, error) {
	<-f.done
	return f.value, f.err
}

func (f *Future[T]) GetWithTimeout(timeout time.Duration) (T, error) {
	select {
	case <-f.done:
		return f.value, f.err
	case <-time.After(timeout):
		var t T
		return t, errors.New("timeout error")
	}
}
