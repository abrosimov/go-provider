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

// Config holds configuration options for the provider library.
//
// Pass this to [Init] during application startup to configure global settings.
//
// See also: [Init].
type Config struct {
	// Logger is used for internal logging. If nil, a no-op logger is used.
	Logger Logger

	// MailboxOutQueueCap is the buffer size for change notification subscriptions.
	//
	// Each subscription has a buffered channel with this capacity. If the buffer
	// fills up, new notifications are dropped.
	//
	// Default: 1 (typically sufficient for production)
	// Higher values (e.g., 10) are useful for tests to avoid dropped notifications.
	//
	// See also: [SubscribeTo], [Subscription].
	MailboxOutQueueCap uint
}

// Init function allows you to redefine some default settings.
//
// IMPORTANT: Init must be called exactly once during application initialization,
// before any goroutines are created and before any other provider operations.
// Calling Init concurrently with other provider operations will cause data races.
//
// Typical usage:
//
//	func main() {
//	    provider.Init(provider.Config{
//	        Logger: myLogger,
//	        MailboxOutQueueCap: 10,
//	    })
//	    // ... rest of initialization ...
//	}
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

// CanProvide checks if type T can be provided. Returns false for interface types.
//
// Interface types cannot be provided because they lack runtime type identity.
// Always provide concrete types instead.
//
// See also: [Provide], [ProvideMultiValue].
func CanProvide[T any]() bool {
	return !IsInterface[T]()
}

// Provide registers a singleton provider for type T.
//
// The createFn will be called exactly once on the first call to [ValueOf][T].
// Subsequent calls to [ValueOf][T] return the cached value. All operations
// are thread-safe using lazy initialization with double-checked locking.
//
// Returns [ErrInterfaceTypeIsNotAllowed] if T is an interface type.
// Returns [ErrProviderAlreadyExists] if a provider for T is already registered.
//
// Example:
//
//	type Database struct {
//		ConnectionString string
//	}
//
//	provider.Provide(func() (*Database, error) {
//		db := &Database{ConnectionString: "postgres://localhost:5432/mydb"}
//		return db, db.Connect()
//	})
//
// See also: [ValueOf], [DoesProvide], [ProvideGuaranteed], [ProvideMultiValue].
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

// GetProvidedTypes returns a list of all registered provider type names.
//
// This function is primarily useful for debugging and introspection.
// The returned slice contains string representations of all types that
// have been registered via [Provide] or [ProvideMultiValue].
//
// See also: [DoesProvide], [DoesProvideMultiValue].
func GetProvidedTypes() []string {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	typeNames := make([]string, 0)
	// TODO: нахуя? Мы просто отдельный список ведём и всё
	defaultRegistry.providers.Range(func(key reflect.Type, value valueProvider) bool {
		typeNames = append(typeNames, value.myUnderlyingTypeIs())
		return true
	})

	return typeNames
}

// ProvideMultiValue registers multiple named providers for type T.
//
// Use this when you need multiple instances of the same type, each identified
// by a unique name. For example, HTTP clients for different API endpoints.
//
// Each [ValueCreator] specifies a name and creation function. The creation
// function is called lazily on the first call to [MultiValueOf] for that name.
//
// Multiple calls to ProvideMultiValue for the same type T will merge the
// named providers. Returns [ErrDuplicateNamedValue] if a name is already registered.
//
// Returns [ErrInterfaceTypeIsNotAllowed] if T is an interface type.
// Returns [ErrTypeIsAlreadyProvided] if T is already provided via [Provide].
//
// Example:
//
//	type HTTPClient struct {
//		BaseURL string
//	}
//
//	provider.ProvideMultiValue(
//		provider.NewDefaultValueCreator("stripe", func() (*HTTPClient, error) {
//			return &HTTPClient{BaseURL: "https://api.stripe.com"}, nil
//		}),
//		provider.NewDefaultValueCreator("github", func() (*HTTPClient, error) {
//			return &HTTPClient{BaseURL: "https://api.github.com"}, nil
//		}),
//	)
//
// See also: [MultiValueOf], [ValueCreator], [NewDefaultValueCreator], [DoesProvideMultiValue].
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

// DoesProvide checks if a singleton provider is registered for type T.
//
// Returns true if [Provide] was called for type T, false otherwise.
// This function does not check multi-value providers; use [DoesProvideMultiValue] for that.
//
// See also: [Provide], [ValueOf], [DoesProvideMultiValue].
func DoesProvide[T any]() bool {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	_, ok := defaultRegistry.providers.Load(getType[T]())
	return ok
}

// DoesProvideMultiValue checks if a multi-value provider is registered for type T.
//
// Returns true if [ProvideMultiValue] was called for type T, false otherwise.
// This function does not check singleton providers; use [DoesProvide] for that.
//
// To check if a specific named value exists, use [DoesProvideNamedMultiValue].
//
// See also: [ProvideMultiValue], [MultiValueOf], [DoesProvide], [DoesProvideNamedMultiValue].
func DoesProvideMultiValue[T any]() bool {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	_, ok := defaultRegistry.multiValueProviders.Load(getType[T]())
	return ok
}

// DoesProvideNamedMultiValue checks if a specific named value exists in a multi-value provider.
//
// Returns true if [ProvideMultiValue] was called for type T with the specified name.
// Returns false if the multi-value provider doesn't exist or the name is not registered.
//
// Example:
//
//	if provider.DoesProvideNamedMultiValue[HTTPClient]("stripe") {
//		client, _ := provider.MultiValueOf[HTTPClient]("stripe")
//		// ... use client ...
//	}
//
// See also: [DoesProvideMultiValue], [MultiValueOf], [ProvideMultiValue].
func DoesProvideNamedMultiValue[T any](name string) bool {
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()

	v, ok := defaultRegistry.multiValueProviders.Load(getType[T]())
	if !ok {
		return false
	}

	mvp := v.(*multiValueProvider[T]) //nolint:errcheck // it's guaranteed that value is comply with `multiValueProvider[T]`.
	mvp.creatorsMtx.Lock()
	defer mvp.creatorsMtx.Unlock()

	_, ok = mvp.creators[name]
	return ok
}

// ValueOf retrieves the singleton value for type T.
//
// On the first call, the creation function registered via [Provide] is invoked.
// Subsequent calls return the cached value. All operations are thread-safe.
//
// Returns [ErrNoProviderForType] if no provider is registered for type T.
// Use [DoesProvide] to check if a provider exists before calling ValueOf.
//
// The returned pointer should not be modified if other goroutines are using it.
// For a safe copy, use [ReadOnlyValueOf] instead.
//
// Example:
//
//	config, err := provider.ValueOf[Config]()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(config.APIKey)
//
// See also: [Provide], [ReadOnlyValueOf], [GuaranteedValueOf], [MultiValueOf].
func ValueOf[T any]() (*T, error) {
	key := getType[T]()

	defaultRegistryMtx.Lock()
	v, ok := defaultRegistry.providers.Load(key)
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

// ReadOnlyValueOf retrieves a copy of the singleton value for type T.
//
// This function returns a copy of the value, making it safe to modify without
// affecting other goroutines. The type T must implement a Copy() method via
// the copier interface constraint.
//
// Returns [ErrNoProviderForType] if no provider is registered for type T.
// Returns [ErrValueIsNilAndNoError] if the provider returned nil without an error.
//
// Example:
//
//	type Config struct {
//		APIKey string
//	}
//
//	func (c *Config) Copy() Config {
//		return Config{APIKey: c.APIKey}
//	}
//
//	config, err := provider.ReadOnlyValueOf[Config, *Config]()
//	// config is a copy, safe to modify
//
// See also: [ValueOf], [ReadOnlyMultiValueOf].
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

// MultiValueOf retrieves a named value from a multi-value provider.
//
// On the first call for a given name, the creation function registered via
// [ProvideMultiValue] is invoked. Subsequent calls for the same name return
// the cached value. All operations are thread-safe.
//
// Returns [ErrNoMultiProviderForType] if no multi-value provider is registered for type T.
// Use [DoesProvideMultiValue] to check if a provider exists before calling MultiValueOf.
//
// The returned pointer should not be modified if other goroutines are using it.
// For a safe copy, use [ReadOnlyMultiValueOf] instead.
//
// Example:
//
//	stripeClient, err := provider.MultiValueOf[HTTPClient]("stripe")
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Use stripeClient...
//
// See also: [ProvideMultiValue], [ReadOnlyMultiValueOf], [DoesProvideNamedMultiValue].
func MultiValueOf[T any](name string) (*T, error) {
	if !CanProvide[T]() {
		return nil, fmt.Errorf("error while providing value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}
	key := getType[T]()

	defaultRegistryMtx.Lock()
	v, ok := defaultRegistry.multiValueProviders.Load(key)
	defaultRegistryMtx.Unlock()

	if !ok {
		return nil, fmt.Errorf("%w %q", ErrNoMultiProviderForType, key)
	}

	mvp := v.(*multiValueProvider[T]) //nolint:errcheck // it's guaranteed that this value is comply with `multiValueProvider[T]`.
	return mvp.value(name)
}

// ReadOnlyMultiValueOf retrieves a copy of a named value from a multi-value provider.
//
// This function returns a copy of the value, making it safe to modify without
// affecting other goroutines. The type T must implement a Copy() method via
// the copier interface constraint.
//
// Returns [ErrNoMultiProviderForType] if no multi-value provider is registered for type T.
// Returns [ErrValueIsNilAndNoError] if the provider returned nil without an error.
//
// Example:
//
//	type Config struct {
//		APIKey string
//	}
//
//	func (c *Config) Copy() Config {
//		return Config{APIKey: c.APIKey}
//	}
//
//	config, err := provider.ReadOnlyMultiValueOf[Config, *Config]("production")
//	// config is a copy, safe to modify
//
// See also: [MultiValueOf], [ReadOnlyValueOf], [ProvideMultiValue].
func ReadOnlyMultiValueOf[T any, V copier[T]](name string) (T, error) {
	var typeInstance T
	if !CanProvide[T]() {
		return typeInstance, fmt.Errorf("error while providing value of %q: %w", GetTypeName[T](), ErrInterfaceTypeIsNotAllowed)
	}

	key := getType[T]()
	defaultRegistryMtx.Lock()
	defer defaultRegistryMtx.Unlock()
	v, ok := defaultRegistry.multiValueProviders.Load(key)

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

// FutureOf creates a future for async value resolution.
//
// A [Future] allows you to wait for a provider to be registered before retrieving
// its value. This is useful for handling initialisation order dependencies where
// one component depends on another that may not be initialised yet.
//
// The future polls every 10ms until [Provide] is called for type T, or the
// context is canceled.
//
// Example:
//
//	future := provider.FutureOf[Config](logger)
//	config, err := future.Get(ctx)  // Waits until Provide[Config]() is called
//	if err != nil {
//		log.Fatal(err)
//	}
//
// See also: [Future], [Provide].
//
//nolint:gocritic // hugeParam is ok here.
func FutureOf[T any](logger zerolog.Logger) Future[T] {
	return newFuture[T](logger)
}

// SubscribeTo creates a subscription for change notifications on type T.
//
// The type T must embed [ChangesNotifier] to enable change notifications.
// When the value calls NotifyListeners(), all subscribers receive a notification
// via their subscription channel.
//
// Subscriptions can be created before or after the provider is registered.
// Multiple subscriptions can exist for the same type.
//
// Important: The subscription channel is buffered with capacity from [Config.MailboxOutQueueCap].
// If the buffer fills up, new notifications are dropped.
//
// Example:
//
//	type AppConfig struct {
//		*provider.ChangesNotifier
//		DebugMode bool
//	}
//
//	subscription := provider.SubscribeTo[AppConfig]()
//	go func() {
//		for range subscription.GetChannel() {
//			config, _ := provider.ValueOf[AppConfig]()
//			fmt.Println("Config changed! DebugMode:", config.DebugMode)
//		}
//	}()
//
// See also: [ChangesNotifier], [Subscription], [SubscribeToNamedValueOf], [UnsubscribeFrom].
func SubscribeTo[T changesNotifier]() *Subscription {
	mbox := getMailboxForType[T]()
	return mbox.GetSubscription()
}

// SubscribeToNamedValueOf creates a subscription for change notifications on a named value.
//
// This is the multi-value variant of [SubscribeTo]. Use this when subscribing to
// a specific named instance from a multi-value provider.
//
// The type T must embed [ChangesNotifier] to enable change notifications.
//
// Example:
//
//	subscription := provider.SubscribeToNamedValueOf[ServerConfig]("production")
//	go func() {
//		for range subscription.GetChannel() {
//			config, _ := provider.MultiValueOf[ServerConfig]("production")
//			fmt.Println("Production config changed")
//		}
//	}()
//
// See also: [SubscribeTo], [MultiValueOf], [ChangesNotifier].
func SubscribeToNamedValueOf[T changesNotifier](name string) *Subscription {
	mbox := getMailboxForNamedValueOfType[T](name)
	return mbox.GetSubscription()
}

// UnsubscribeFrom removes a subscription from the notification system.
//
// After unsubscribing, the subscription will no longer receive notifications.
// It is safe to call this function multiple times with the same subscription.
//
// Note: This function is typically not needed as subscriptions are automatically
// cleaned up when the registry is reset via [ResetRegistry].
//
// See also: [SubscribeTo], [SubscribeToNamedValueOf], [Subscription].
func UnsubscribeFrom[T any](listener *Subscription) {
	// TODO: would it be better, to add Subscription.Destroy() method instead of
	//    having this api call.
	if !IsChangesNotifier[T]() {
		return
	}

	mbox := getMailboxForType[T]()
	mbox.Unsubscribe(listener)
}
