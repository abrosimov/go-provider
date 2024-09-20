package provider_test

import "github.com/abrosimov/go-provider"

type mySafeType struct {
	name string
}

func (*mySafeType) IGuaranteeSafeBehaviour() {
	//
}

func (s *ProviderTestSuite) TestProvideGuaranteed() {
	provider.ProvideGuaranteed[mySafeType](func() *mySafeType {
		return &mySafeType{name: "test"}
	})

	obj := provider.GuaranteedValueOf[mySafeType]()
	s.Require().Equal("test", obj.name)
}

func (s *ProviderTestSuite) TestProvideGuaranteedDoubleWontWork() {
	provider.ProvideGuaranteed(func() *mySafeType {
		return &mySafeType{name: "test"}
	})

	obj := provider.GuaranteedValueOf[mySafeType]()
	s.Require().Equal("test", obj.name)

	provider.ProvideGuaranteed(func() *mySafeType {
		return &mySafeType{name: "test2"}
	})
	obj = provider.GuaranteedValueOf[mySafeType]()
	s.Require().Equal("test", obj.name)
}

type mySafeMultiValueType struct {
	name string
}

func (m *mySafeMultiValueType) IGuaranteeSafeBehaviour() {
	//
}

func (s *ProviderTestSuite) TestProvideMultiValueGuaranteed() {
	mvp1 := provider.NewDefaultGuaranteedValueCreator("golang", func(s string) *mySafeMultiValueType {
		return &mySafeMultiValueType{name: s}
	})
	mvp2 := provider.NewDefaultGuaranteedValueCreator("php", func(s string) *mySafeMultiValueType {
		return &mySafeMultiValueType{name: s}
	})

	list := provider.NewGuaranteedValueCreatorsList[mySafeMultiValueType](2)
	list = append(list, mvp1, mvp2)

	provider.ProvideMultiValueGuaranteed[mySafeMultiValueType](list...)

	v1 := provider.GuaranteedMultiValueOf[mySafeMultiValueType]("golang")
	s.Require().Equal("golang", v1.name)
	v2 := provider.GuaranteedMultiValueOf[mySafeMultiValueType]("php")
	s.Require().Equal("php", v2.name)

	v3 := provider.GuaranteedMultiValueOf[mySafeMultiValueType]("not-known")
	s.Require().Nil(v3)
}
