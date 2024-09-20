package provider

type ValueCreator[T any] interface {
	Name() string
	Create() (*T, error)
}

type DefaultValueCreator[T any] struct {
	createFn CreateFn[T]
	name     string
}

// NewDefaultValueCreator creates a new value creator that could be used as parameter for newMultiValueProvider.
func NewDefaultValueCreator[T any](name string, createFn CreateFn[T]) DefaultValueCreator[T] {
	return DefaultValueCreator[T]{
		name:     name,
		createFn: createFn,
	}
}

// Name returns the name of the value creator.
func (d DefaultValueCreator[T]) Name() string {
	return d.name
}

// Create returns a new instance of the value.
func (d DefaultValueCreator[T]) Create() (*T, error) {
	return d.createFn()
}

type GuaranteedValueCreator[T any, V SafetyGuarantor[T]] interface {
	Name() string
	Create() *T
}

type DefaultGuaranteedValueCreator[T any, V SafetyGuarantor[T]] struct {
	createFn CreateMultiGuaranteedFn[T, V]
	name     string
}

func NewDefaultGuaranteedValueCreator[T any, V SafetyGuarantor[T]](
	name string, creator CreateMultiGuaranteedFn[T, V],
) DefaultGuaranteedValueCreator[T, V] {
	return DefaultGuaranteedValueCreator[T, V]{
		name:     name,
		createFn: creator,
	}
}

func (d DefaultGuaranteedValueCreator[T, V]) Name() string {
	return d.name
}

func (d DefaultGuaranteedValueCreator[T, V]) Create() *T {
	return d.createFn(d.name)
}
