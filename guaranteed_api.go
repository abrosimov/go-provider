package provider

// SafetyGuarantor is a marker interface that guarantees that the provided value will never be nil.
type SafetyGuarantor[T any] interface {
	// IGuaranteeSafeBehaviour is a marker interface that guarantees
	// that the function provided for object creation will never return an error, and it's result never won't be a nil.
	IGuaranteeSafeBehaviour()

	// Because provider operates with pointers, we can't just add IGuaranteeSafeBehaviour
	// to the provided type as pointer receiver so attempt to call ProvideGuaranteed[T]
	// will fail with "Type does not implement SafetyGuarantor[T] as the IGuaranteeSafeBehaviour method has a pointer receiver" error
	//
	// So we also embed *T to the SafetyGuarantor[T] interface to make it work.
	// More details are here: https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#pointer-method-example
	*T
}

// ProvideGuaranteed is a helper function that provides a value of the given type.
// The value is guaranteed to be created and returned without an error.
// If the value cannot be created, the expected behaviour is undefined.
func ProvideGuaranteed[T any, V SafetyGuarantor[T]](createFn CreateGuaranteedFn[T, V]) {
	if !CanProvide[T]() {
		return
	}

	stdCreateFn := func() (*T, error) {
		return createFn(), nil
	}

	_ = Provide[T](stdCreateFn)
}

// GuaranteedValueOf is a helper function that returns a value of the given type.
// The value is guaranteed to be created and returned without an error.
// If the value cannot be created, the expected behaviour is undefined.
func GuaranteedValueOf[T any, _ SafetyGuarantor[T]]() *T {
	v, _ := ValueOf[T]()
	return v
}

func GuaranteedReadOnlyValueOf[T any, _ SafetyGuarantor[T], V copier[T]]() T {
	v, _ := ReadOnlyValueOf[T, V]()
	return v
}

func GuaranteedMultiValueOf[T any, _ SafetyGuarantor[T]](name string) *T {
	v, _ := MultiValueOf[T](name)
	return v
}

func GuaranteedReadOnlyMultiValueOf[T any, _ SafetyGuarantor[T], V copier[T]](name string) T {
	v, _ := ReadOnlyMultiValueOf[T, V](name)
	return v
}

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
