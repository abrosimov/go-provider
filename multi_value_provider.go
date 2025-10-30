package provider

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	ErrDuplicateNamedValue    = errors.New("duplicate named value")
	ErrNoNamedValueInProvider = errors.New("no named value in provider")
)

type valueHandler[T any] struct {
	creator  ValueCreator[T]
	provider *provider[T]
	once     sync.Once
}

type multiValueProvider[T any] struct {
	creators    map[string]*valueHandler[T]
	providers   map[string]*provider[T]
	providedAt  string
	creatorsMtx sync.Mutex
}

func newMultiValueProvider[T any](vc ...ValueCreator[T]) (*multiValueProvider[T], error) {
	names := make(map[string]struct{}, len(vc))
	creators := make(map[string]*valueHandler[T], len(vc))
	for _, c := range vc {
		if _, ok := names[c.Name()]; ok {
			var t T
			return nil, fmt.Errorf("%w: %q for %T", ErrDuplicateNamedValue, c.Name(), t)
		}

		names[c.Name()] = struct{}{}
		creators[c.Name()] = &valueHandler[T]{
			creator: c,
		}
	}
	mvp := &multiValueProvider[T]{
		creators:   creators,
		providers:  make(map[string]*provider[T], len(vc)),
		providedAt: getStackTrace(),
	}

	return mvp, nil
}

// merge merges two multiValueProvider[T] instances into one.
//
//nolint:unused // idk why linter thinks that this function is unused
func (mvp *multiValueProvider[T]) merge(mvp2 *multiValueProvider[T]) error {
	mvp.creatorsMtx.Lock()
	defer mvp.creatorsMtx.Unlock()

	for name, creator := range mvp2.creators {
		if _, ok := mvp.creators[name]; ok {
			return fmt.Errorf("%w: %q for multiValueProvider[%T]", ErrDuplicateNamedValue, name, mvp.myUnderlyingTypeIs())
		}

		mvp.creators[name] = creator
	}

	return nil
}

func (mvp *multiValueProvider[T]) value(name string) (*T, error) {
	mvp.creatorsMtx.Lock()
	defer mvp.creatorsMtx.Unlock()

	creator, ok := mvp.creators[name]
	if !ok {
		var t T
		return nil, fmt.Errorf("%w: %q for %T", ErrNoNamedValueInProvider, name, t)
	}

	creator.once.Do(func() {
		mvp.creators[name].provider = newProvider[T](creator.creator.Create)
	})

	return mvp.creators[name].provider.value()
}

// iamMultiValueProviderOf returns the reflect.Type of multiValueProvider[T].
func (mvp *multiValueProvider[T]) iAmMultiValueProviderOf() reflect.Type {
	return getType[T]()
}

// myUnderlyingTypeIs returns the string representation of the name of the underlying type of multiValueProvider[T].
func (mvp *multiValueProvider[T]) myUnderlyingTypeIs() string {
	return GetTypeName[T]()
}
