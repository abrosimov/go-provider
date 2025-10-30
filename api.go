package provider

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/rs/zerolog"
)

var (
	ErrInterfaceTypeIsNotAllowed           = errors.New("interfaces are not allowed to be registered as provided types")
	ErrNoProviderForType                   = errors.New("no provider for type")
	ErrNoMultiProviderForType              = errors.New("no multi-provider for type")
	ErrValueIsNilAndNoError                = errors.New("value is nil and there is no error")
	ErrAttemptToSubscribeToNonNotifierType = errors.New("attempt to subscribe to non-notifier type")
)

type Config struct {
	// MailboxOutQueueCap is the maximum number of changes that should be kept.
	// Roughly speaking, this option is required only during unit tests. In the normal
	// life out queue cap is 1, which is enough.
	Logger             Logger
	MailboxOutQueueCap uint
}

// Init function allows you to redefine some default settings.
func Init(config Config) {
	if config.Logger != nil {
		logger = config.Logger
	}

	if config.MailboxOutQueueCap == 0 {
		logger.Warnf("passed MailboxOutQueueCap is equal to zero, switching value to default %d", defaultOutboxCap)
		config.MailboxOutQueueCap = defaultOutboxCap
	}

	outBoxCap = config.MailboxOutQueueCap
}

func CanProvide[T any]() bool {
	return !IsInterface[T]()
}

func Provide[T any](createFn CreateFn[T]) error {
	if !CanProvide[T]() {
		return fmt.Errorf("error while providing value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}

	p := newProvider(createFn)
	if err := addProviderToRegistry(p); err != nil {
		return err
	}
	return nil
}

func GetProvidedTypes() []string {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	typeNames := make([]string, 0)
	// TODO: нахуя? Мы просто отдельный список ведём и всё
	defaultRegistry.Load().providers.Range(func(key reflect.Type, value valueProvider) bool {
		typeNames = append(typeNames, value.myUnderlyingTypeIs())
		return true
	})

	return typeNames
}

func ProvideMultiValue[T any](vc ...ValueCreator[T]) error {
	if !CanProvide[T]() {
		return fmt.Errorf("error while providing multi-value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}

	mvp, err := newMultiValueProvider(vc...)
	if err != nil {
		return err
	}

	if err := addMultiValueProviderToRegistry[T](mvp); err != nil {
		return err
	}

	return nil
}

func DoesProvide[T any]() bool {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	_, ok := defaultRegistry.Load().providers.Load(getType[T]())
	return ok
}

func DoesProvideMultiValue[T any]() bool {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	_, ok := defaultRegistry.Load().multiValueProviders.Load(getType[T]())
	return ok
}

func DoesProvideNamedMultiValue[T any](name string) bool {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	v, ok := defaultRegistry.Load().multiValueProviders.Load(getType[T]())
	if !ok {
		return false
	}

	mvp := v.(*multiValueProvider[T]) //nolint:errcheck // it's guaranteed that value is comply with `multiValueProvider[T]`.
	mvp.creatorsMtx.Lock()
	defer mvp.creatorsMtx.Unlock()

	_, ok = mvp.creators[name]
	return ok
}

func ValueOf[T any]() (*T, error) {
	key := getType[T]()

	defaultRegistryMtx.Lock()
	v, ok := defaultRegistry.Load().providers.Load(key)
	defaultRegistryMtx.Unlock()

	if !ok {
		return nil, fmt.Errorf("%w %q", ErrNoProviderForType, key)
	}

	pr, ok := v.(*provider[T])
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrNoProviderForType, key)
	}

	return pr.value()
}

type copier[T any] interface {
	Copy() T
	*T
}

func ReadOnlyValueOf[T any, V copier[T]]() (T, error) {
	var typeInstance T
	if !CanProvide[T]() {
		return typeInstance, fmt.Errorf("error while providing value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}

	val, err := ValueOf[T]()
	if err != nil {
		return typeInstance, err
	}

	if val == nil {
		return typeInstance, ErrValueIsNilAndNoError
	}

	// Here we have `ptr` which is `*T`. `T` doesn't have `Copy()` method, but `V` does.
	// We need to cast `*T` to `V`, which has `*T` in its definition. But we can't do
	// just ptr.(V) because `ptr` is just a `*T`. So first we need to cast `*T` to `any`,
	// and then we're able to cast to `V`, which also means that we can call `Copy()`.
	copierPtr := any(val).(V) //nolint:errcheck // it's guaranteed that `T` is a `Copier[T]`.
	return copierPtr.Copy(), err
}

func MultiValueOf[T any](name string) (*T, error) {
	if !CanProvide[T]() {
		return nil, fmt.Errorf("error while providing value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}
	key := getType[T]()

	defaultRegistryMtx.Lock()
	v, ok := defaultRegistry.Load().multiValueProviders.Load(key)
	defaultRegistryMtx.Unlock()

	if !ok {
		return nil, fmt.Errorf("%w %q", ErrNoMultiProviderForType, key)
	}

	mvp := v.(*multiValueProvider[T]) //nolint:errcheck // it's guaranteed that this value is comply with `multiValueProvider[T]`.
	return mvp.value(name)
}

func ReadOnlyMultiValueOf[T any, V copier[T]](name string) (T, error) {
	var typeInstance T
	if !CanProvide[T]() {
		return typeInstance, fmt.Errorf("error while providing value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}

	key := getType[T]()
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()
	v, ok := defaultRegistry.Load().multiValueProviders.Load(key)

	if !ok {
		return typeInstance, fmt.Errorf("%w %q", ErrNoMultiProviderForType, key)
	}

	mvp := v.(*multiValueProvider[T]) //nolint:errcheck // it's guaranteed that this value is comply with `valueProvider`.
	val, err := mvp.value(name)
	if err != nil {
		return typeInstance, err
	}

	if val == nil {
		return typeInstance, ErrValueIsNilAndNoError
	}

	// Here we have `ptr` which is `*T`. `T` doesn't have `Copy()` method, but `V` does.
	// We need to cast `*T` to `V`, which has `*T` in its definition. But we can't do
	// just ptr.(V) because `ptr` is just a `*T`. So first we need to cast `*T` to `any`,
	// and then we're able to cast to `V`, which also means that we can call `Copy()`.
	copierPtr := any(val).(V) //nolint:errcheck // it's guaranteed that `T` is a `Copier[T]`.
	return copierPtr.Copy(), err
}

type changesNotifier interface {
	NotifyListeners() error
}

// FutureOf creates a new future object.
//
//nolint:gocritic // hugeParam is ok here.
func FutureOf[T any](logger zerolog.Logger) Future[T] {
	return newFuture[T](logger)
}

// SubscribeTo creates a Subscription object for the given type.
func SubscribeTo[T changesNotifier]() *Subscription {
	mbox := getMailboxForType[T]()
	return mbox.GetSubscription()
}

func SubscribeToNamedValueOf[T changesNotifier](name string) *Subscription {
	mbox := getMailboxForNamedValueOfType[T](name)
	return mbox.GetSubscription()
}

func UnsubscribeFrom[T any](listener *Subscription) {
	// TODO: would it be better, to add Subscription.Destroy() method instead of
	//    having this api call.
	if !IsChangesNotifier[T]() {
		return
	}

	mbox := getMailboxForType[T]()
	mbox.Unsubscribe(listener)
}
