package provider

import (
	"fmt"
	"sync/atomic"
)

// Subscription represents a subscription to value change notifications.
//
// Created via [SubscribeTo] or [SubscribeToNamedValueOf], a Subscription provides
// a channel that receives notifications when the subscribed value changes.
//
// The notification channel is buffered with capacity from [Config.MailboxOutQueueCap].
// If the buffer fills up, new notifications are dropped (non-blocking behaviour).
//
// Example:
//
//	subscription := provider.SubscribeTo[AppConfig]()
//	go func() {
//		for range subscription.GetChannel() {
//			config, _ := provider.ValueOf[AppConfig]()
//			fmt.Println("Config changed:", config)
//		}
//	}()
//
// See also: [SubscribeTo], [SubscribeToNamedValueOf], [ChangesNotifier].
type Subscription struct {
	changesStream chan struct{}
	isValid       atomic.Bool
}

func newSubscription() *Subscription {
	s := &Subscription{
		changesStream: make(chan struct{}, outBoxCap),
	}

	s.isValid.Store(true)
	return s
}

// GetChannel returns the notification channel.
//
// Use this channel in a for-range loop to receive change notifications.
// The channel emits a value each time [ChangesNotifier.NotifyListeners] is called.
//
// The channel is closed when the registry is reset via [ResetRegistry].
//
// See also: [IsValid], [SubscribeTo].
func (s *Subscription) GetChannel() <-chan struct{} {
	return s.changesStream
}

// IsValid returns whether the subscription is still active.
//
// A subscription becomes invalid when the registry is reset via [ResetRegistry],
// which closes the notification channel.
//
// See also: [GetChannel], [ResetRegistry].
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
