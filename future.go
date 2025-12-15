package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/rs/zerolog"
)

var (
	ErrFutureWaitTimedOut = errors.New("timeout while waiting for")
)

const (
	defaultSleepTimeWhileWaiting = time.Millisecond * 10
)

// Future enables async value resolution for initialisation order dependencies.
//
// A Future allows you to wait for a provider to be registered before retrieving
// its value. This is useful when component A depends on component B, but B may
// not be initialised yet.
//
// Create a Future using [FutureOf]. The Future polls every 10ms until [Provide]
// is called for type T, or the context/timeout expires.
//
// Example:
//
//	// Component that depends on Config
//	go func() {
//		future := provider.FutureOf[Config](logger)
//		config, err := future.Get(ctx)  // Waits until Provide[Config]() is called
//		if err != nil {
//			log.Fatal(err)
//		}
//		// Now use config to initialise this component
//	}()
//
//	// Later, Config is provided
//	provider.Provide(func() (*Config, error) {
//		return &Config{...}, nil
//	})
//
// See also: [FutureOf], [Provide].
type Future[T any] struct {
	expectedType reflect.Type
	logger       zerolog.Logger
}

//nolint:gocritic // hugeParam is ok here.
func newFuture[T any](logger zerolog.Logger) Future[T] {
	return Future[T]{
		logger: logger.With().
			Str("subsystem", "provider::future").
			Str("type_parameter", GetTypeName[T]()).
			Logger(),
		expectedType: getType[T](),
	}
}

// Wait blocks until the provider for type T is registered.
//
// This method polls every 10ms until [Provide] is called for type T.
// It blocks indefinitely - use [Future.WaitFor] or [Future.Get] if you need a timeout.
//
// See also: [Future.WaitFor], [Future.Get], [FutureOf].
func (f Future[T]) Wait() {
	for {
		if DoesProvide[T]() {
			return
		}

		time.Sleep(defaultSleepTimeWhileWaiting)
	}
}

// WaitFor blocks until the provider for type T is registered or the timeout expires.
//
// This method polls every 10ms until [Provide] is called for type T or the timeout
// duration elapses. Returns [ErrFutureWaitTimedOut] if the timeout is reached.
//
// For context-based cancellation, use [Future.Get] instead.
//
// See also: [Future.Wait], [Future.Get], [FutureOf].
func (f Future[T]) WaitFor(timeout time.Duration) error {
	stopTime := time.Now().Add(timeout)
	for time.Now().Before(stopTime) {
		if DoesProvide[T]() {
			return nil
		}

		f.logger.Debug().Msgf("Waiting for %q to be provided", GetTypeName[T]())
		time.Sleep(defaultSleepTimeWhileWaiting)
	}

	f.logger.Error().Msgf("Timeout while waiting for %q", GetTypeName[T]())
	return fmt.Errorf("%w Future[%q]", ErrFutureWaitTimedOut, GetTypeName[T]())
}

// Get waits for the provider to be registered and returns the value.
//
// This method polls every 10ms until [Provide] is called for type T, then retrieves
// and returns the value via [ValueOf]. If the context is canceled before the provider
// is registered, returns the context error.
//
// This is the most common way to use a Future, as it combines waiting with value retrieval.
//
// Example:
//
//	future := provider.FutureOf[Config](logger)
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	config, err := future.Get(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Use config...
//
// See also: [Future.Wait], [Future.WaitFor], [FutureOf], [ValueOf].
func (f Future[T]) Get(ctx context.Context) (*T, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context for Future[%v] canceled: %w", GetTypeName[T](), ctx.Err())
		default:
			value, err := ValueOf[T]()
			if err != nil {
				if errors.Is(err, ErrNoProviderForType) {
					time.Sleep(defaultSleepTimeWhileWaiting)
					f.logger.Debug().Msgf("Waiting for %q to be provided for return", GetTypeName[T]())
					continue
				}

				return nil, fmt.Errorf("error %w happened while getting value from provider[%s]", err, GetTypeName[T]())
			}

			return value, nil
		}
	}
}
