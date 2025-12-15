package provider

import (
	"errors"
	"fmt"
)

var (
	ErrChangeNotifierDestroyed = errors.New("change notifier has been destroyed")
)

type changeSender interface {
	send() error
}

// ChangesNotifier enables pub/sub change notifications for provided values.
//
// Embed this struct into your types to enable change notifications. When the value
// changes, call [ChangesNotifier.NotifyListeners] to notify all subscribers.
//
// Create a ChangesNotifier using [NewChangesNotifier] for singleton providers or
// [NewMultiValueChangesNotifier] for multi-value providers.
//
// Example:
//
//	type AppConfig struct {
//		*provider.ChangesNotifier
//		DebugMode bool
//	}
//
//	func (c *AppConfig) SetDebugMode(enabled bool) {
//		c.DebugMode = enabled
//		c.NotifyListeners()  // Notify all subscribers
//	}
//
//	provider.Provide(func() (*AppConfig, error) {
//		return &AppConfig{
//			ChangesNotifier: provider.NewChangesNotifier[AppConfig](),
//			DebugMode: false,
//		}, nil
//	})
//
// See also: [NewChangesNotifier], [SubscribeTo], [Subscription].
type ChangesNotifier struct {
	mbox   changeSender
	logger Logger
	name   string
}

// NewChangesNotifier creates a ChangesNotifier for singleton providers.
//
// Use this when creating a singleton value that will be provided via [Provide].
// Embed the returned ChangesNotifier in your struct to enable change notifications.
//
// Example:
//
//	type AppConfig struct {
//		*provider.ChangesNotifier
//		DebugMode bool
//	}
//
//	provider.Provide(func() (*AppConfig, error) {
//		return &AppConfig{
//			ChangesNotifier: provider.NewChangesNotifier[AppConfig](),
//			DebugMode: false,
//		}, nil
//	})
//
// See also: [ChangesNotifier], [NewMultiValueChangesNotifier], [SubscribeTo].
func NewChangesNotifier[T any]() *ChangesNotifier {
	return &ChangesNotifier{
		name: GetTypeName[T](),
		mbox: getMailboxForType[T](),
	}
}

// NewMultiValueChangesNotifier creates a ChangesNotifier for multi-value providers.
//
// Use this when creating a named value that will be provided via [ProvideMultiValue].
// The name parameter should match the name used in ProvideMultiValue.
//
// Example:
//
//	type ServerConfig struct {
//		*provider.ChangesNotifier
//		MaxConnections int
//	}
//
//	provider.ProvideMultiValue(
//		provider.NewDefaultValueCreator("production", func() (*ServerConfig, error) {
//			return &ServerConfig{
//				ChangesNotifier: provider.NewMultiValueChangesNotifier[ServerConfig]("production"),
//				MaxConnections: 1000,
//			}, nil
//		}),
//	)
//
// See also: [ChangesNotifier], [NewChangesNotifier], [SubscribeToNamedValueOf].
func NewMultiValueChangesNotifier[T any](name string) *ChangesNotifier {
	return &ChangesNotifier{
		name:   fmt.Sprintf("%s@%s", GetTypeName[T](), name),
		mbox:   getMailboxForNamedValueOfType[T](name),
		logger: logger,
	}
}

// NotifyListeners sends a notification to all subscribers.
//
// Call this method whenever the value changes to notify all subscribers created
// via [SubscribeTo] or [SubscribeToNamedValueOf].
//
// Notifications are sent asynchronously through buffered channels. If a subscriber's
// channel is full, the notification is dropped (not blocking).
//
// Example:
//
//	type AppConfig struct {
//		*provider.ChangesNotifier
//		DebugMode bool
//	}
//
//	func (c *AppConfig) SetDebugMode(enabled bool) {
//		c.DebugMode = enabled
//		c.NotifyListeners()  // Notify all subscribers of the change
//	}
//
// See also: [SubscribeTo], [ChangesNotifier].
func (cn *ChangesNotifier) NotifyListeners() error {
	return cn.mbox.send()
}

func (cn *ChangesNotifier) Copy() *ChangesNotifier {
	return &ChangesNotifier{
		name:   cn.name,
		mbox:   &noopMailbox{},
		logger: cn.logger,
	}
}
