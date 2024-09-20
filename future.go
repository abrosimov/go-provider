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

func (f Future[T]) Wait() {
	for {
		if DoesProvide[T]() {
			return
		}

		time.Sleep(defaultSleepTimeWhileWaiting)
	}
}

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

func (f Future[T]) Get(ctx context.Context) (*T, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context for Future[%v] calceled: %w", GetTypeName[T](), ctx.Err())
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
