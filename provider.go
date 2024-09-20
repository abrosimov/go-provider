package provider

import (
	"fmt"
	"reflect"
	"sync"

	"errors"
)

var (
	ErrFailedToEvaluateValue = errors.New("failed to evaluate value")
)

type provider[T any] struct {
	createFn   CreateFn[T]
	val        *T
	providedAt string
	valMtx     sync.Mutex
}

// newProvider creates a new provider that will create the value when the provider is created.
func newProvider[T any](createFn CreateFn[T]) *provider[T] {
	return &provider[T]{
		createFn:   createFn,
		providedAt: getStackTrace(),
	}
}

// Value returns the value of the provider. If the value is nil, it will be created.
func (p *provider[T]) value() (*T, error) {
	p.valMtx.Lock()
	if p.val != nil {
		defer p.valMtx.Unlock()
		return p.val, nil
	}
	p.valMtx.Unlock()

	p.valMtx.Lock()
	defer p.valMtx.Unlock()

	// Check p.value again, because it could have been created by another goroutine
	// while we were waiting for the `p.valMtx.Lock()`.
	if p.val != nil {
		return p.val, nil
	}

	value, err := p.createFn()
	if err != nil {
		return nil, fmt.Errorf("%w for %T with error: %w", ErrFailedToEvaluateValue, p.val, err)
	}

	p.val = value
	return p.val, nil
}

func (*provider[T]) iAmProviderOf() reflect.Type {
	return getType[T]()
}

func (*provider[T]) myUnderlyingTypeIs() string {
	return GetTypeName[T]()
}
