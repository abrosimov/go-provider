/*
Package provider implements a type-safe dependency injection library using Go generics.

# Overview

go-provider enables lazy initialization of singleton values with thread-safe creation,
multi-value providers for named instances of the same type, change notifications for
reactive updates, and futures for async value resolution.

# Quick Start

Register a provider and retrieve the value:

	type Config struct {
		APIKey  string
		Timeout int
	}

	provider.Provide(func() (*Config, error) {
		return &Config{APIKey: "secret", Timeout: 30}, nil
	})

	config, err := provider.ValueOf[Config]()

The creation function is called exactly once on the first call to [ValueOf].
Subsequent calls return the cached value.

# Architecture

The library consists of three main components:

  - [Registry]: Global singleton managing all providers and subscriptions
  - Provider: Lazy-initialised singleton wrapper for type T (created via [Provide])
  - Multi-value provider: Multiple named instances of same type (via [ProvideMultiValue])

All operations are thread-safe using mutex-based synchronisation.

# Initialization

Call [Init] once at application startup before any goroutines are created:

	func main() {
		provider.Init(provider.Config{
			Logger:             myLogger,
			MailboxOutQueueCap: 10,
		})
		// ... rest of initialization ...
	}

See [Config] for configuration options. Calling [Init] concurrently with other
provider operations will cause data races.

# Multi-Value Providers

Use multi-value providers to manage multiple named instances of the same type,
such as HTTP clients for different destinations:

	type HTTPClient struct {
		BaseURL string
	}

	provider.ProvideMultiValue(
		provider.DefaultValueCreator[HTTPClient]{
			name: "stripe",
			createFn: func() (*HTTPClient, error) {
				return &HTTPClient{BaseURL: "https://api.stripe.com"}, nil
			},
		},
		provider.DefaultValueCreator[HTTPClient]{
			name: "github",
			createFn: func() (*HTTPClient, error) {
				return &HTTPClient{BaseURL: "https://api.github.com"}, nil
			},
		},
	)

	stripeClient, _ := provider.MultiValueOf[HTTPClient]("stripe")

See [ProvideMultiValue], [MultiValueOf], and [ValueCreator] for details.

# Change Notifications

Types can embed [ChangesNotifier] to enable pub/sub notifications when values change:

	type AppConfig struct {
		*provider.ChangesNotifier
		DebugMode bool
	}

	func (c *AppConfig) SetDebugMode(enabled bool) {
		c.DebugMode = enabled
		c.NotifyListeners()
	}

	// Subscribe to changes
	subscription := provider.SubscribeTo[AppConfig]()
	go func() {
		for range subscription.GetChannel() {
			config, _ := provider.ValueOf[AppConfig]()
			fmt.Println("Config changed! DebugMode:", config.DebugMode)
		}
	}()

See [SubscribeTo], [ChangesNotifier], and [Subscription] for details.

# Futures

Futures enable async value resolution for handling initialization order dependencies:

	func initDatabase(ctx context.Context) {
		future := provider.FutureOf[Config](logger)
		config, err := future.Get(ctx)  // Waits until Provide[Config]() is called
		// ... use config to initialise database ...
	}

See [FutureOf] and [Future] for details.

# Guaranteed API

For values that are guaranteed to exist and never fail, use the guaranteed API
with compile-time safety:

	type AppName struct {
		Value string
	}

	func (*AppName) IGuaranteeSafeBehaviour() {}

	provider.ProvideGuaranteed[AppName, *AppName](func() *AppName {
		return &AppName{Value: "MyApp"}
	})

	appName := provider.GuaranteedValueOf[AppName, *AppName]()

The [SafetyGuarantor] interface ensures only types explicitly marked as safe can
use the guaranteed API. See [ProvideGuaranteed] and [GuaranteedValueOf] for details.

# Type Constraints

Interfaces cannot be provided as they lack runtime type identity:

	provider.Provide(func() (io.Reader, error) { ... })  // Compile error

Always provide concrete types and use [CanProvide] to check at runtime if needed.

# Testing

Use [ResetRegistry] to clean state between tests:

	func TestMyCode(t *testing.T) {
		provider.ResetRegistry()

		provider.Provide(func() (*Config, error) {
			return &Config{APIKey: "test-key"}, nil
		})

		// ... test code ...
	}

# Error Handling

The library defines several sentinel errors:

  - [ErrNoProviderForType]: ValueOf called before Provide
  - [ErrProviderAlreadyExists]: Provide called twice for same type
  - [ErrInterfaceTypeIsNotAllowed]: Attempted to provide interface type
  - [ErrDuplicateNamedValue]: Multi-value provider has duplicate name
  - [ErrTypeIsAlreadyProvided]: Mixing single/multi-value providers for same type

All errors can be checked using errors.Is.
*/
package provider
