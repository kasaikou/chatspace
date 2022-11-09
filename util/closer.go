package util

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	ErrAlreadyClosed = errors.New("already closed")
)

type channelCloser[T interface{}] struct {
	closed bool
	lock   sync.Mutex
	ch     chan<- T
}

func (ch *channelCloser[T]) Close() error {
	if ch.closed {
		return fmt.Errorf("channel %w", ErrAlreadyClosed)
	}

	ch.lock.Lock()
	defer ch.lock.Unlock()
	close(ch.ch)
	ch.closed = true

	return nil
}

func WrapChannelCloser[T interface{}](ch chan<- T) io.Closer {
	return &channelCloser[T]{
		closed: false,
		ch:     ch,
	}
}
