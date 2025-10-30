package provider_test

import (
	"reflect"

	"github.com/abrosimov/go-provider"
)

func (s *ProviderTestSuite) TestCanProvide() {
	s.Require().True(provider.CanProvide[MyGlobalObject]())
	s.Require().True(provider.CanProvide[MyGlobalGeneric[int]]())
	s.Require().False(provider.CanProvide[MyGlobalInterface]())
}

type myLocalObject struct {
	name string
}

func (m *myLocalObject) Copy() myLocalObject {
	return myLocalObject{name: m.name}
}

func (s *ProviderTestSuite) TestValueOfStruct() {
	err := provider.Provide(
		func() (*myLocalObject, error) {
			return &myLocalObject{name: "test"}, nil
		})
	s.Require().NoError(err, "failed to provide value")

	// Ensure that calling Value() multiple times returns the same instance
	obj1, err := provider.ValueOf[myLocalObject]()
	s.Require().NoError(err)
	obj2, err := provider.ValueOf[myLocalObject]()
	s.Require().NoError(err)
	s.Require().Same(obj1, obj2)

	// Ensure that ReadOnly version has the same value, but results are different instances.
	obj3, err := provider.ReadOnlyValueOf[myLocalObject]()
	s.Require().NoError(err)
	s.Require().NotSame(obj1, obj3)
	s.Require().NotSame(obj2, obj3)
	s.Require().Equal(obj1.name, obj3.name)

	s.Require().True(provider.DoesProvide[myLocalObject]())
}

type myNotProvidedLocalObject struct{}

func (m *myNotProvidedLocalObject) Copy() myNotProvidedLocalObject {
	return myNotProvidedLocalObject{}
}

func (s *ProviderTestSuite) TestValueOfNotProvidedType() {
	obj4, err := provider.ValueOf[myNotProvidedLocalObject]()
	s.Require().Error(err)
	s.Require().Nil(obj4)

	obj5, err := provider.ReadOnlyValueOf[myNotProvidedLocalObject]()
	s.Require().Error(err)
	s.Require().Equal(myNotProvidedLocalObject{}, obj5)
}

type myLocalGeneric[T any] struct {
	val T
}

func (m *myLocalGeneric[T]) Copy() myLocalGeneric[T] {
	return myLocalGeneric[T]{val: m.val}
}

func (s *ProviderTestSuite) TestValueOfOfGeneric() {
	err := provider.Provide(
		func() (*myLocalGeneric[int], error) {
			return &myLocalGeneric[int]{val: 42}, nil
		})
	s.Require().NoError(err, "failed to provide value")

	// Ensure that calling Value() multiple times returns the same instance
	obj1, err := provider.ValueOf[myLocalGeneric[int]]()
	s.Require().NoError(err)
	obj2, err := provider.ValueOf[myLocalGeneric[int]]()
	s.Require().NoError(err)
	s.Require().Same(obj1, obj2)

	// Ensure that ReadOnly version has the same value, but results are different instances.
	obj3, err := provider.ReadOnlyValueOf[myLocalGeneric[int]]()
	s.Require().NoError(err)
	s.Require().NotSame(obj1, obj3)
	s.Require().NotSame(obj2, obj3)
	s.Require().Equal(obj1.val, obj3.val)
}

func (s *ProviderTestSuite) TestValueOfGenericForDifferentTypes() {
	type myLocalGeneric[T any] struct {
		val T
	}

	err := provider.Provide(
		func() (*myLocalGeneric[int], error) {
			return &myLocalGeneric[int]{val: 42}, nil
		})
	s.Require().NoError(err, "failed to provide value")

	err = provider.Provide(
		func() (*myLocalGeneric[int64], error) {
			return &myLocalGeneric[int64]{val: 420}, nil
		})
	s.Require().NoError(err, "failed to provide value")

	// Ensure that calling Value() multiple times returns the same instance
	obj1, err := provider.ValueOf[myLocalGeneric[int]]()
	s.Require().NoError(err)
	obj2, err := provider.ValueOf[myLocalGeneric[int64]]()
	s.Require().NoError(err)
	s.Require().NotSame(obj1, obj2)
}

func (s *ProviderTestSuite) TestValueOfInterface() {
	/*
		We don't allow providing values through the interface types.
		The reason is pretty simple: there could be several implementations of the same interface,
		so we have no guarantee that the registered underlying type will be the same to expected one.
	*/
	type myLocalInterface interface{}
	err := provider.Provide[myLocalInterface](func() (*myLocalInterface, error) {
		return nil, nil
	})

	s.Require().ErrorIs(err, provider.ErrInterfaceTypeIsNotAllowed)
}

func (s *ProviderTestSuite) TestMultiValueOfStruct() {
	creators := make([]provider.ValueCreator[MyGlobalObject], 0)
	for _, v := range []string{"test1", "test2", "test3"} {
		creators = append(creators, valueCreatorFactory(v))
	}
	err := provider.ProvideMultiValue[MyGlobalObject](creators...)
	s.Require().NoError(err)

	s.Require().True(provider.DoesProvideMultiValue[MyGlobalObject]())
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test1"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test2"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test3"))
	s.Require().False(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test4"))
	s.Require().False(provider.DoesProvideNamedMultiValue[MyGlobalGeneric[int]]("test4"))

	obj1, err := provider.MultiValueOf[MyGlobalObject]("test1")
	s.Require().NoError(err)
	s.Require().NotNil(obj1.name)
	s.Require().Equal("test1", obj1.name)

	obj2, err := provider.MultiValueOf[MyGlobalObject]("test2")
	s.Require().NoError(err)
	s.Require().NotNil(obj2.name)
	s.Require().Equal("test1", obj1.name)

	obj3, err := provider.MultiValueOf[MyGlobalObject]("test3")
	s.Require().NoError(err)
	s.Require().NotNil(obj3)
	s.Require().Equal("test1", obj1.name)

	obj4, err := provider.MultiValueOf[MyGlobalObject]("test4")
	s.Require().Error(err)
	s.Require().Nil(obj4)
}

func (s *ProviderTestSuite) TestMultiValueMergeCreators() {
	firstCreators := make([]provider.ValueCreator[MyGlobalObject], 0)
	secondCreators := make([]provider.ValueCreator[MyGlobalObject], 0)
	for _, v := range []string{"test1", "test2", "test3"} {
		firstCreators = append(firstCreators, valueCreatorFactory(v))
	}
	for _, v := range []string{"test5", "test6", "test7"} {
		secondCreators = append(secondCreators, valueCreatorFactory(v))
	}
	err := provider.ProvideMultiValue[MyGlobalObject](firstCreators...)
	s.Require().NoError(err)
	err = provider.ProvideMultiValue[MyGlobalObject](secondCreators...)
	s.Require().NoError(err)

	s.Require().True(provider.DoesProvideMultiValue[MyGlobalObject]())
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test1"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test2"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test3"))
	s.Require().False(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test4"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test5"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test6"))
	s.Require().True(provider.DoesProvideNamedMultiValue[MyGlobalObject]("test7"))
}

func (s *ProviderTestSuite) TestMultiValueCantCreateWithDuplicates() {
	firstCreators := make([]provider.ValueCreator[MyGlobalObject], 0)
	for _, v := range []string{"test1", "test2", "test1"} {
		firstCreators = append(firstCreators, valueCreatorFactory(v))
	}
	err := provider.ProvideMultiValue[MyGlobalObject](firstCreators...)
	s.Require().ErrorIs(err, provider.ErrDuplicateNamedValue)
}

func (s *ProviderTestSuite) TestMultiValueCantMergeWithDuplicates() {
	firstCreators := make([]provider.ValueCreator[MyGlobalObject], 0)
	for _, v := range []string{"test1", "test2", "test3"} {
		firstCreators = append(firstCreators, valueCreatorFactory(v))
	}
	err := provider.ProvideMultiValue[MyGlobalObject](firstCreators...)
	s.Require().NoError(err)

	secondCreators := make([]provider.ValueCreator[MyGlobalObject], 0)
	for _, v := range []string{"test1"} {
		secondCreators = append(secondCreators, valueCreatorFactory(v))
	}

	err = provider.ProvideMultiValue[MyGlobalObject](secondCreators...)
	s.Require().ErrorIs(err, provider.ErrDuplicateNamedValue)
}

func (s *ProviderTestSuite) TestMultiValueCantProvideInterface() {
	err := provider.ProvideMultiValue[any](
		provider.NewDefaultValueCreator[any]("name", func() (*any, error) {
			r := &MyGlobalObject{name: "test"}
			vp := reflect.New(reflect.TypeOf(r))
			vp.Elem().Set(reflect.ValueOf(r))
			i := vp.Interface()
			return &i, nil
		}))
	s.Require().ErrorIs(err, provider.ErrInterfaceTypeIsNotAllowed)
}

func (s *ProviderTestSuite) TestProvideFailsIfAlreadyExists() {
	err := provider.Provide[MyGlobalObject](func() (*MyGlobalObject, error) {
		return &MyGlobalObject{name: "test"}, nil
	})
	s.Require().NoError(err)
	err = provider.Provide[MyGlobalObject](func() (*MyGlobalObject, error) {
		return &MyGlobalObject{name: "test"}, nil
	})
	s.Require().ErrorIs(err, provider.ErrProviderAlreadyExists)
}

func valueCreatorFactory(name string) provider.DefaultValueCreator[MyGlobalObject] {
	return provider.NewDefaultValueCreator[MyGlobalObject](name, func() (*MyGlobalObject, error) {
		return &MyGlobalObject{name: name}, nil
	})
}
