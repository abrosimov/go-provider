package provider

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

var (
	ErrProviderAlreadyExists           = errors.New("provider already exists")
	ErrMultiValueProviderAlreadyExists = errors.New("multi-value provider already exists")
	ErrTypeIsAlreadyProvided           = errors.New("type is already provided with type")
)

var (
	defaultRegistry    = atomic.Pointer[Registry]{}
	defaultRegistryMtx = sync.Mutex{}
)

func init() {
	defaultRegistry.Store(&Registry{name: "default"})
}

type providerType string

const (
	providerTypeSingleValue providerType = "single-value provider"
	providerTypeMultiValue  providerType = "multi-value provider"
)

type valueProvider interface {
	iAmProviderOf() reflect.Type
	myUnderlyingTypeIs() string
}

type multipleValueProvider[T any] interface {
	iAmMultiValueProviderOf() reflect.Type
	myUnderlyingTypeIs() string
	merge(mvp2 *multiValueProvider[T]) error
}

type Registry struct {
	name string
	//nolint:godox // silence
	// TODO: very naive approach, need to benchmark on it's performance
	// 	map[reflect.Type]valueProvider might be more efficient?
	providers           sync.Map
	multiValueProviders sync.Map
	registeredTypes     sync.Map
	mailboxes           sync.Map
	// providedTypes       []string
}

type CreateFn[T any] func() (*T, error)
type CreateGuaranteedFn[T any, V SafetyGuarantor[T]] func() *T
type CreateMultiGuaranteedFn[T any, V SafetyGuarantor[T]] func(string) *T

func NewRegistry(name string) *Registry {
	return &Registry{name: name}
}

func (r *Registry) destroy() {
	r.providers.Clear()
	r.mailboxes.Clear()
	r.registeredTypes.Clear()
	r.multiValueProviders.Clear()
}

// ResetRegistry resets the default registry to a new empty registry.
// In most cases, this function should be called in a test setup function.
func ResetRegistry() error {
	defaultRegistryMtx.Lock()
	oldRegistry := defaultRegistry.Load()
	defaultRegistry.Store(&Registry{name: "default"})
	defaultRegistryMtx.Unlock()

	oldRegistry.destroy()

	return nil
}

func addProviderToRegistry(provider valueProvider) error {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	providerTypeVal, loaded := defaultRegistry.Load().registeredTypes.LoadOrStore(provider.iAmProviderOf(), providerTypeSingleValue)
	if loaded && providerTypeVal != providerTypeSingleValue {
		return fmt.Errorf("%s %w via %s", provider.myUnderlyingTypeIs(), ErrTypeIsAlreadyProvided, providerTypeVal)
	}

	_, loaded = defaultRegistry.Load().providers.LoadOrStore(
		provider.iAmProviderOf(),
		provider,
	)
	if loaded {
		return fmt.Errorf("%w for type %s", ErrProviderAlreadyExists, provider.myUnderlyingTypeIs())
	}

	return nil
}

func getMailboxForType[T any]() *Mailbox {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	mbox := NewMailbox(GetTypeName[T]())
	v, _ := defaultRegistry.Load().mailboxes.LoadOrStore(
		mbox.name,
		mbox,
	)

	return v.(*Mailbox)
}

func getMailboxForNamedValueOfType[T any](name string) *Mailbox {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	mbox := NewMailbox(fmt.Sprintf("%s@%s", GetTypeName[T](), name))
	v, _ := defaultRegistry.Load().mailboxes.LoadOrStore(
		mbox.name,
		mbox,
	)

	return v.(*Mailbox)
}

func addMultiValueProviderToRegistry[T any](provider multipleValueProvider[T]) error {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	providerTypeVal, loaded := defaultRegistry.Load().registeredTypes.LoadOrStore(provider.iAmMultiValueProviderOf(), providerTypeMultiValue)
	if loaded && providerTypeVal != providerTypeMultiValue {
		return fmt.Errorf("%s %w via %s", provider.myUnderlyingTypeIs(), ErrTypeIsAlreadyProvided, providerTypeVal)
	}

	mvpRaw, loaded := defaultRegistry.Load().multiValueProviders.LoadOrStore(
		provider.iAmMultiValueProviderOf(),
		provider,
	)

	if loaded {
		return mvpRaw.(multipleValueProvider[T]).merge(provider.(*multiValueProvider[T]))
	}

	return nil
}
