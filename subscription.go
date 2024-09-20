package provider

import (
	"fmt"
	"sync/atomic"
)

type Subscription struct {
	changesStream chan struct{}
	isValid       atomic.Bool
}

func newSubscription() *Subscription {
	const defaultOutBoxCap = 10 // WTF? Oo
	s := &Subscription{
		changesStream: make(chan struct{}, defaultOutBoxCap),
	}

	s.isValid.Store(true)
	return s
}

func (s *Subscription) GetChannel() <-chan struct{} {
	return s.changesStream
}

func (s *Subscription) IsValid() bool {
	return s.isValid.Load()
}

func (s *Subscription) invalidate() error {
	if !s.isValid.CompareAndSwap(true, false) {
		return fmt.Errorf("%w of Subscription", ErrDoubleDestroy)
	}
	close(s.changesStream)
	return nil
}
