package provider

// SafetyGuarantor is a marker interface for types that guarantee safe, error-free value creation.
//
// Types implementing this interface can use the guaranteed API ([ProvideGuaranteed],
// [GuaranteedValueOf], etc.) which provides cleaner code by eliminating error handling.
//
// The interface requires:
//  1. IGuaranteeSafeBehaviour() method - marker that the creation function never returns
//     nil and never fails
//  2. Embedding *T - required due to Go's pointer receiver semantics with generics
//
// Example:
//
//	type AppName struct {
//		Value string
//	}
//
//	func (*AppName) IGuaranteeSafeBehaviour() {}
//
//	// Now AppName can use guaranteed API
//	provider.ProvideGuaranteed[AppName, *AppName](func() *AppName {
//		return &AppName{Value: "MyApp"}
//	})
//
// Technical note: The *T embedding is required because the provider library operates
// with pointers. Without it, types with pointer receivers would fail the interface
// constraint. See: https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#pointer-method-example
//
// See also: [ProvideGuaranteed], [GuaranteedValueOf].
type SafetyGuarantor[T any] interface {
	// IGuaranteeSafeBehaviour is a marker method that guarantees the creation function
	// will never return an error and its result will never be nil.
	IGuaranteeSafeBehaviour()

	// Embedding *T is required for types with pointer receivers to satisfy the interface.
	*T
}

// ProvideGuaranteed registers a singleton provider with compile-time safety guarantees.
//
// This is a variant of [Provide] for types implementing [SafetyGuarantor], which
// guarantees at compile-time that:
//  1. The createFn will never return nil
//  2. The createFn will never fail (no error return value)
//
// IMPORTANT: This function silently ignores errors. If called with an interface type
// or if the provider already exists, the registration will fail silently.
// Use the regular [Provide] function if you need error feedback.
//
// Use this API when you have compile-time guarantees that value creation cannot fail,
// such as for configuration constants or statically-initialised values.
//
// Example:
//
//	type AppName struct {
//		Value string
//	}
//
//	func (*AppName) IGuaranteeSafeBehaviour() {}
//
//	provider.ProvideGuaranteed[AppName, *AppName](func() *AppName {
//		return &AppName{Value: "MyApp"}
//	})
//
// See also: [GuaranteedValueOf], [SafetyGuarantor], [Provide], [ProvideMultiValueGuaranteed].
func ProvideGuaranteed[T any, V SafetyGuarantor[T]](createFn CreateGuaranteedFn[T, V]) {
	if !CanProvide[T]() {
		return
	}

	stdCreateFn := func() (*T, error) {
		return createFn(), nil
	}

	_ = Provide[T](stdCreateFn)
}

// GuaranteedValueOf retrieves a singleton value without error handling.
//
// This is a variant of [ValueOf] for types implementing [SafetyGuarantor].
// It ignores all errors and returns nil if the provider doesn't exist or creation fails.
//
// IMPORTANT: Always call [ProvideGuaranteed] before calling GuaranteedValueOf.
// If you need error handling, use the regular [ValueOf] function instead.
//
// Use this API when you have runtime guarantees that the provider exists and will succeed,
// such as in application code after all initialization is complete.
//
// Example:
//
//	// During initialization
//	provider.ProvideGuaranteed[AppName, *AppName](func() *AppName {
//		return &AppName{Value: "MyApp"}
//	})
//
//	// Later in application code
//	appName := provider.GuaranteedValueOf[AppName, *AppName]()
//	fmt.Println(appName.Value)  // No error handling needed
//
// See also: [ProvideGuaranteed], [SafetyGuarantor], [ValueOf], [GuaranteedMultiValueOf].
func GuaranteedValueOf[T any, _ SafetyGuarantor[T]]() *T {
	v, _ := ValueOf[T]()
	return v
}

// GuaranteedReadOnlyValueOf retrieves a copy of a singleton value without error handling.
//
// This is a variant of [ReadOnlyValueOf] for types implementing [SafetyGuarantor].
// It returns a copy of the value, making it safe to modify. Errors are ignored.
//
// The type T must implement a Copy() method via the copier interface.
//
// See also: [GuaranteedValueOf], [ReadOnlyValueOf], [GuaranteedReadOnlyMultiValueOf].
func GuaranteedReadOnlyValueOf[T any, _ SafetyGuarantor[T], V copier[T]]() T {
	v, _ := ReadOnlyValueOf[T, V]()
	return v
}

// GuaranteedMultiValueOf retrieves a named value without error handling.
//
// This is a variant of [MultiValueOf] for types implementing [SafetyGuarantor].
// It ignores all errors and returns nil if the provider doesn't exist or creation fails.
//
// IMPORTANT: Always call [ProvideMultiValueGuaranteed] before calling GuaranteedMultiValueOf.
// If you need error handling, use the regular [MultiValueOf] function instead.
//
// See also: [ProvideMultiValueGuaranteed], [GuaranteedValueOf], [MultiValueOf].
func GuaranteedMultiValueOf[T any, _ SafetyGuarantor[T]](name string) *T {
	v, _ := MultiValueOf[T](name)
	return v
}

// GuaranteedReadOnlyMultiValueOf retrieves a copy of a named value without error handling.
//
// This is a variant of [ReadOnlyMultiValueOf] for types implementing [SafetyGuarantor].
// It returns a copy of the value, making it safe to modify. Errors are ignored.
//
// The type T must implement a Copy() method via the copier interface.
//
// See also: [GuaranteedMultiValueOf], [ReadOnlyMultiValueOf], [GuaranteedReadOnlyValueOf].
func GuaranteedReadOnlyMultiValueOf[T any, _ SafetyGuarantor[T], V copier[T]](name string) T {
	v, _ := ReadOnlyMultiValueOf[T, V](name)
	return v
}

// ProvideMultiValueGuaranteed registers multiple named providers with compile-time safety guarantees.
//
// This is a variant of [ProvideMultiValue] for types implementing [SafetyGuarantor].
// Each [GuaranteedValueCreator] specifies a name and creation function that never fails.
//
// IMPORTANT: This function silently ignores errors. If called with an interface type
// or if registration fails, it will fail silently. Use [ProvideMultiValue] if you need
// error feedback.
//
// Example:
//
//	type Config struct {
//		Name string
//	}
//
//	func (*Config) IGuaranteeSafeBehaviour() {}
//
//	provider.ProvideMultiValueGuaranteed(
//		provider.NewDefaultGuaranteedValueCreator("dev", func(name string) *Config {
//			return &Config{Name: "DevConfig"}
//		}),
//		provider.NewDefaultGuaranteedValueCreator("prod", func(name string) *Config {
//			return &Config{Name: "ProdConfig"}
//		}),
//	)
//
// See also: [GuaranteedMultiValueOf], [ProvideMultiValue], [ProvideGuaranteed], [GuaranteedValueCreator].
func ProvideMultiValueGuaranteed[T any, V SafetyGuarantor[T]](vc ...GuaranteedValueCreator[T, V]) {
	if !CanProvide[T]() {
		return
	}

	stdValueCreators := make([]ValueCreator[T], 0, len(vc))
	st := make([]GuaranteedValueCreator[T, V], 0)
	_ = st
	for _, creator := range vc {
		stdValueCreators = append(stdValueCreators,
			DefaultValueCreator[T]{
				name: creator.Name(),
				createFn: func() (*T, error) {
					return creator.Create(), nil
				},
			},
		)
	}

	_ = ProvideMultiValue[T](stdValueCreators...)
}

// NewGuaranteedValueCreatorsList creates a list that can hold elements of GuaranteedValueCreator.
// The main reason for existence of this function is a fact, that golang type inference isn't ideal in this particular case.
// For example, we can't create such list by simple expression make([]provider.GuaranteedValueCreator[MyGuaranteedType], 0),
// and we can't do it with make([]provider.GuaranteedValueCreator[MyGuaranteedType, MyGuaranteedType], 0).
// So the only way for us is to create this helper function.
func NewGuaranteedValueCreatorsList[T any, V SafetyGuarantor[T]](capacity int) []GuaranteedValueCreator[T, V] {
	return make([]GuaranteedValueCreator[T, V], 0, capacity)
}
