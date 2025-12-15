package provider

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrDoubleDestroy                 = errors.New("double destroy")
	ErrAttemptToSendToInvalidMailbox = errors.New("attempt to send to invalid mailbox")
)

type mailbox struct {
	logger      Logger
	queueIn     chan struct{}
	name        string
	queueOut    []*Subscription
	queueOutMtx sync.Mutex
	isValid     atomic.Bool
	hasChanged  atomic.Bool
}

func newMailbox(name string) *mailbox {
	m := &mailbox{
		name:     name,
		queueIn:  make(chan struct{}),
		queueOut: make([]*Subscription, 0, outBoxCap),
		logger:   logger,
	}
	m.isValid.Store(true)

	go m.mainLoop()
	return m
}

func (m *mailbox) Len() int {
	m.queueOutMtx.Lock()
	defer m.queueOutMtx.Unlock()

	return len(m.queueOut)
}

func (m *mailbox) GetSubscription() *Subscription {
	m.queueOutMtx.Lock()
	defer m.queueOutMtx.Unlock()

	s := newSubscription()

	m.queueOut = append(m.queueOut, s)
	// TODO: let's set index of s into subscription. Type name would be also great.
	// Also it means that we'll be able to unsubscribe in one step.
	if m.hasChanged.Load() {
		// if there was at least one change - let's notify new subscription and allow it asap update its state.
		s.changesStream <- struct{}{}
	}

	return s
}

func (m *mailbox) Unsubscribe(s *Subscription) {
	m.queueOutMtx.Lock()
	defer m.queueOutMtx.Unlock()

	for i := range m.queueOut {
		if m.queueOut[i] == s {
			close(m.queueOut[i].changesStream)
			m.queueOut = append(m.queueOut[:i], m.queueOut[i+1:]...)
			return
		}
	}
}

func (m *mailbox) destroy() error {
	if !m.isValid.CompareAndSwap(true, false) {
		m.logger.Warnf("attempt to destroy already destroyed mailbox")
		return ErrDoubleDestroy
	}

	close(m.queueIn)

	m.queueOutMtx.Lock()
	defer m.queueOutMtx.Unlock()
	for i := range m.queueOut {
		err := m.queueOut[i].invalidate()
		if err != nil {
			m.logger.Warnf("error while invalidating subscription")
		}
	}
	m.queueOut = nil
	return nil
}

func (m *mailbox) IsValid() bool {
	return m.isValid.Load()
}

func (m *mailbox) send() error {
	if !m.IsValid() {
		return ErrAttemptToSendToInvalidMailbox
	}

	m.queueIn <- struct{}{}
	m.hasChanged.CompareAndSwap(false, true)
	return nil
}

func (m *mailbox) mainLoop() {
	for range m.queueIn {
		m.broadcastChanges()
	}
}

func (m *mailbox) broadcastChanges() {
	m.queueOutMtx.Lock()
	defer m.queueOutMtx.Unlock()

	// just a ring buffer to prevent blocking when we try to notify slow listeners
	var wg sync.WaitGroup
	wg.Add(len(m.queueOut))
	for i := range m.queueOut {
		go func(idx int) {
			defer wg.Done()
			select {
			case m.queueOut[idx].changesStream <- struct{}{}:
			default:
				<-m.queueOut[idx].changesStream
				m.queueOut[idx].changesStream <- struct{}{}
			}
		}(i)
	}
	wg.Wait()
}

type noopMailbox struct{}

func (n *noopMailbox) send() error {
	return nil
}
