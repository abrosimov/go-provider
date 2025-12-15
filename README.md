# go-provider

A type-safe dependency injection library for Go using generics. Provides lazy initialization, multi-value providers, change notifications, and futures for async value resolution.

## Features

- **Type-safe DI** - Compile-time type safety using Go generics
- **Lazy initialization** - Thread-safe singleton creation on first access
- **Multi-value providers** - Multiple named instances of the same type
- **Change notifications** - React to value changes via pub/sub
- **Futures** - Async value resolution for initialization order dependencies
- **Guaranteed API** - Compile-time safety guarantees for infallible operations

## Installation

```bash
go get github.com/abrosimov/go-provider
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/abrosimov/go-provider"
)

type Config struct {
    APIKey string
    Timeout int
}

func main() {
    // Register a provider
    provider.Provide(func() (*Config, error) {
        return &Config{
            APIKey: "secret",
            Timeout: 30,
        }, nil
    })

    // Get the value (lazy initialization on first call)
    config, err := provider.ValueOf[Config]()
    if err != nil {
        panic(err)
    }

    fmt.Println(config.APIKey) // "secret"
}
```

## Usage Examples

### Basic Provider

```go
type Database struct {
    ConnectionString string
}

// Register provider
err := provider.Provide(func() (*Database, error) {
    db := &Database{ConnectionString: "postgres://..."}
    // Perform initialization
    return db, nil
})

// Access value (thread-safe singleton)
db, err := provider.ValueOf[Database]()
```

### Multi-Value Providers

Perfect for managing multiple instances of the same type (e.g., HTTP clients per destination):

```go
type HTTPClient struct {
    BaseURL string
    // ... other fields
}

// Register multiple HTTP clients
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

// Access by name
stripeClient, _ := provider.MultiValueOf[HTTPClient]("stripe")
githubClient, _ := provider.MultiValueOf[HTTPClient]("github")
```

### Change Notifications

React to value changes using the pub/sub system:

```go
type AppConfig struct {
    *provider.ChangesNotifier  // Embed to enable notifications

    DebugMode bool
}

func (c *AppConfig) SetDebugMode(enabled bool) {
    c.DebugMode = enabled
    c.NotifyListeners()  // Notify subscribers
}

// Subscribe to changes
func main() {
    provider.Provide(func() (*AppConfig, error) {
        return &AppConfig{
            ChangesNotifier: provider.NewChangesNotifier[AppConfig](),
            DebugMode: false,
        }, nil
    })

    // Subscribe
    subscription := provider.SubscribeTo[AppConfig]()

    go func() {
        for range subscription.GetChannel() {
            config, _ := provider.ValueOf[AppConfig]()
            fmt.Println("Config changed! DebugMode:", config.DebugMode)
        }
    }()

    // Trigger change
    config, _ := provider.ValueOf[AppConfig]()
    config.SetDebugMode(true)  // Subscribers will be notified
}
```

### Futures (Async Resolution)

Handle initialization order dependencies:

```go
func initDatabase(ctx context.Context) {
    // Wait for config to be provided
    future := provider.FutureOf[Config](logger)
    config, err := future.Get(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Now initialize database with config
    db := &Database{ConnectionString: config.DBUrl}
    provider.Provide(func() (*Database, error) {
        return db, nil
    })
}
```

### Guaranteed API

For values that are guaranteed to exist and never fail:

```go
type AppName struct {
    Value string
}

// Implement SafetyGuarantor to opt into guaranteed API
func (*AppName) IGuaranteeSafeBehaviour() {}

// Register with guaranteed API (no error handling)
provider.ProvideGuaranteed[AppName, *AppName](func() *AppName {
    return &AppName{Value: "MyApp"}
})

// Access without error handling (returns nil if not registered)
appName := provider.GuaranteedValueOf[AppName, *AppName]()
fmt.Println(appName.Value)
```

## Configuration

Call `Init()` once at application startup to configure global settings:

```go
func main() {
    provider.Init(provider.Config{
        Logger: myLogger,
        MailboxOutQueueCap: 10,  // Subscription buffer size (default: 1)
    })

    // ... rest of initialization
}
```

**Important:** `Init()` must be called before any goroutines are created and before any provider operations.

## Testing

The library provides `ResetRegistry()` for tests:

```go
func TestMyCode(t *testing.T) {
    // Clean slate for each test
    provider.ResetRegistry()

    // Register test providers
    provider.Provide(func() (*Config, error) {
        return &Config{APIKey: "test-key"}, nil
    })

    // Test code...
}
```

## Architecture

- **Registry** - Global singleton holding all providers
- **Lazy initialization** - Values created on first access using double-checked locking
- **Thread-safe** - All operations protected by mutexes and sync.Maps
- **Type constraints** - Interfaces cannot be provided (compile-time enforcement)

## Error Handling

The library defines several sentinel errors:

- `ErrNoProviderForType` - ValueOf called before Provide
- `ErrProviderAlreadyExists` - Provide called twice for same type
- `ErrInterfaceTypeIsNotAllowed` - Attempted to provide interface type
- `ErrDuplicateNamedValue` - Multi-value provider has duplicate name
- `ErrTypeIsAlreadyProvided` - Mixing single/multi-value providers for same type

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions welcome! Please ensure:
- All tests pass (`go test ./...`)
- Linter passes (`golangci-lint run`)
- Code coverage remains high
