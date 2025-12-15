package provider_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/abrosimov/go-provider"
)

// Example demonstrates basic provider usage: register a provider and retrieve the value.
func Example() {
	_ = provider.ResetRegistry()

	type Config struct {
		APIKey string
	}

	// Register a provider
	_ = provider.Provide(func() (*Config, error) {
		return &Config{APIKey: "secret"}, nil
	})

	// Get the value (lazy initialization)
	config, _ := provider.ValueOf[Config]()
	fmt.Println(config.APIKey)
	// Output: secret
}

// ExampleProvide shows how to register a singleton provider for a type.
func ExampleProvide() {
	_ = provider.ResetRegistry()

	type Database struct {
		Connected bool
	}

	// Register provider - creation function called once on first access
	err := provider.Provide(func() (*Database, error) {
		db := &Database{Connected: true}
		return db, nil
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Provider registered successfully")
	// Output: Provider registered successfully
}

// ExampleValueOf demonstrates retrieving a singleton value.
func ExampleValueOf() {
	_ = provider.ResetRegistry()

	type Config struct {
		Timeout int
	}

	_ = provider.Provide(func() (*Config, error) {
		return &Config{Timeout: 30}, nil
	})

	// First call initialises the value
	config, err := provider.ValueOf[Config]()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Timeout:", config.Timeout)
	// Output: Timeout: 30
}

// ExampleProvideMultiValue shows how to register multiple named instances of the same type.
func ExampleProvideMultiValue() {
	_ = provider.ResetRegistry()

	type APIClient struct {
		Name string
	}

	err := provider.ProvideMultiValue(
		provider.NewDefaultValueCreator("client-a", func() (*APIClient, error) {
			return &APIClient{Name: "ClientA"}, nil
		}),
		provider.NewDefaultValueCreator("client-b", func() (*APIClient, error) {
			return &APIClient{Name: "ClientB"}, nil
		}),
	)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Multi-value provider registered")
	// Output: Multi-value provider registered
}

// ExampleMultiValueOf demonstrates retrieving a named instance from a multi-value provider.
func ExampleMultiValueOf() {
	_ = provider.ResetRegistry()

	type HTTPClient struct {
		BaseURL string
	}

	_ = provider.ProvideMultiValue(
		provider.NewDefaultValueCreator("stripe", func() (*HTTPClient, error) {
			return &HTTPClient{BaseURL: "https://api.stripe.com"}, nil
		}),
		provider.NewDefaultValueCreator("github", func() (*HTTPClient, error) {
			return &HTTPClient{BaseURL: "https://api.github.com"}, nil
		}),
	)

	// Retrieve by name
	stripeClient, _ := provider.MultiValueOf[HTTPClient]("stripe")
	githubClient, _ := provider.MultiValueOf[HTTPClient]("github")

	fmt.Println("Stripe:", stripeClient.BaseURL)
	fmt.Println("GitHub:", githubClient.BaseURL)
	// Output:
	// Stripe: https://api.stripe.com
	// GitHub: https://api.github.com
}

// ExampleSubscribeTo demonstrates subscribing to value changes.
func ExampleSubscribeTo() {
	_ = provider.ResetRegistry()

	type AppConfig struct {
		*provider.ChangesNotifier
		DebugMode bool
	}

	_ = provider.Provide(func() (*AppConfig, error) {
		return &AppConfig{
			ChangesNotifier: provider.NewChangesNotifier[AppConfig](),
			DebugMode:       false,
		}, nil
	})

	// Subscribe to changes
	subscription := provider.SubscribeTo[AppConfig]()

	// Listen for changes in background
	done := make(chan bool)
	go func() {
		<-subscription.GetChannel()
		config, _ := provider.ValueOf[AppConfig]()
		fmt.Println("DebugMode changed to:", config.DebugMode)
		done <- true
	}()

	// Trigger a change
	config, _ := provider.ValueOf[AppConfig]()
	config.DebugMode = true
	_ = config.NotifyListeners()

	<-done
	// Output: DebugMode changed to: true
}

// ExampleFutureOf demonstrates waiting for a provider to be registered asynchronously.
func ExampleFutureOf() {
	_ = provider.ResetRegistry()

	type Config struct {
		APIKey string
	}

	logger := zerolog.Nop()

	// Simulate async initialization
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = provider.Provide(func() (*Config, error) {
			return &Config{APIKey: "async-key"}, nil
		})
	}()

	// Wait for the provider to be registered
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	future := provider.FutureOf[Config](logger)
	config, err := future.Get(ctx)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("APIKey:", config.APIKey)
	// Output: APIKey: async-key
}

// AppName is a type that implements SafetyGuarantor for guaranteed API example.
type AppName struct {
	Value string
}

// IGuaranteeSafeBehaviour implements SafetyGuarantor interface.
func (*AppName) IGuaranteeSafeBehaviour() {}

// ExampleProvideGuaranteed shows the guaranteed API for error-free value creation.
func ExampleProvideGuaranteed() {
	_ = provider.ResetRegistry()

	// Register with guaranteed API (no error handling)
	provider.ProvideGuaranteed[AppName, *AppName](func() *AppName {
		return &AppName{Value: "MyApp"}
	})

	// Access without error handling
	appName := provider.GuaranteedValueOf[AppName, *AppName]()
	fmt.Println("App:", appName.Value)
	// Output: App: MyApp
}

// Example_multipleHTTPClients demonstrates a real-world use case of multi-value providers
// for managing HTTP clients to different API endpoints.
func Example_multipleHTTPClients() {
	_ = provider.ResetRegistry()

	type HTTPClient struct {
		Name    string
		BaseURL string
		Timeout int
	}

	// Register multiple HTTP clients
	_ = provider.ProvideMultiValue(
		provider.NewDefaultValueCreator("payment", func() (*HTTPClient, error) {
			return &HTTPClient{
				Name:    "PaymentAPI",
				BaseURL: "https://api.payment.com",
				Timeout: 30,
			}, nil
		}),
		provider.NewDefaultValueCreator("analytics", func() (*HTTPClient, error) {
			return &HTTPClient{
				Name:    "AnalyticsAPI",
				BaseURL: "https://api.analytics.com",
				Timeout: 10,
			}, nil
		}),
		provider.NewDefaultValueCreator("notifications", func() (*HTTPClient, error) {
			return &HTTPClient{
				Name:    "NotificationAPI",
				BaseURL: "https://api.notifications.com",
				Timeout: 5,
			}, nil
		}),
	)

	// Use different clients for different purposes
	paymentClient, _ := provider.MultiValueOf[HTTPClient]("payment")
	analyticsClient, _ := provider.MultiValueOf[HTTPClient]("analytics")

	fmt.Printf("Payment: %s (timeout: %ds)\n", paymentClient.BaseURL, paymentClient.Timeout)
	fmt.Printf("Analytics: %s (timeout: %ds)\n", analyticsClient.BaseURL, analyticsClient.Timeout)
	// Output:
	// Payment: https://api.payment.com (timeout: 30s)
	// Analytics: https://api.analytics.com (timeout: 10s)
}

// Example_changeNotifications demonstrates a complete pub/sub pattern using ChangesNotifier.
func Example_changeNotifications() {
	_ = provider.ResetRegistry()
	type ServerConfig struct {
		*provider.ChangesNotifier
		logLevel       string
		mu             sync.RWMutex
		maxConnections int
	}
	getState := func(cfg *ServerConfig) (int, string) {
		cfg.mu.RLock()
		defer cfg.mu.RUnlock()
		return cfg.maxConnections, cfg.logLevel
	}
	updateConfig := func(cfg *ServerConfig, maxConn int, logLevel string) {
		cfg.mu.Lock()
		cfg.maxConnections, cfg.logLevel = maxConn, logLevel
		cfg.mu.Unlock()
		_ = cfg.NotifyListeners()
	}
	_ = provider.Provide(func() (*ServerConfig, error) {
		return &ServerConfig{
			ChangesNotifier: provider.NewChangesNotifier[ServerConfig](),
			maxConnections:  100,
			logLevel:        "info",
		}, nil
	})
	subscription := provider.SubscribeTo[ServerConfig]()
	done, changeCount := make(chan bool), 0
	go func() {
		for range subscription.GetChannel() {
			changeCount++
			config, _ := provider.ValueOf[ServerConfig]()
			maxConn, logLevel := getState(config)
			fmt.Printf("Change %d: MaxConnections=%d, LogLevel=%s\n", changeCount, maxConn, logLevel)
			if changeCount == 2 {
				done <- true
				return
			}
		}
	}()
	config, _ := provider.ValueOf[ServerConfig]()
	updateConfig(config, 200, "debug")
	time.Sleep(10 * time.Millisecond)
	updateConfig(config, 500, "warn")
	<-done
	// Output:
	// Change 1: MaxConnections=200, LogLevel=debug
	// Change 2: MaxConnections=500, LogLevel=warn
}

// Example_initializationOrder demonstrates using futures to handle initialization dependencies.
func Example_initializationOrder() {
	_ = provider.ResetRegistry()

	type Config struct {
		DatabaseURL string
	}

	type Database struct {
		URL       string
		Connected bool
	}

	logger := zerolog.Nop()

	// Database initialisation depends on Config
	// Using Future to wait for Config to be provided
	go func() {
		future := provider.FutureOf[Config](logger)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		config, err := future.Get(ctx)
		if err != nil {
			fmt.Println("Error waiting for config:", err)
			return
		}

		// Now we can initialise database with config
		_ = provider.Provide(func() (*Database, error) {
			return &Database{
				URL:       config.DatabaseURL,
				Connected: true,
			}, nil
		})
	}()

	// Config is provided later
	time.Sleep(20 * time.Millisecond)
	_ = provider.Provide(func() (*Config, error) {
		return &Config{DatabaseURL: "postgres://localhost:5432/mydb"}, nil
	})

	// Wait for database to be initialised
	time.Sleep(50 * time.Millisecond)

	db, _ := provider.ValueOf[Database]()
	fmt.Printf("Database connected: %v\n", db.Connected)
	fmt.Printf("Database URL: %s\n", db.URL)
	// Output:
	// Database connected: true
	// Database URL: postgres://localhost:5432/mydb
}
