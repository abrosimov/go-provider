package provider_test

/*
func (s *ProviderTestSuite) TestMultiValueProvider() {
	type testStruct struct {
		Name string
	}

	s.Run("Register with duplicate name", func() {
		mvp, _ := provider.newMultiValueProvider[testStruct](newTestValueCreator("test", &testStruct{Name: "test"}))

		err := mvp.Register(newTestValueCreator("test", &testStruct{Name: "test"}))

		s.Require().Error(err)
		s.Require().True(errors.Is(err, provider.ErrDuplicateNamedValue))
	})

	s.Run("Value with invalid name", func() {
		mvp, _ := provider.newMultiValueProvider[testStruct](newTestValueCreator("test", &testStruct{Name: "test"}))

		_, err := mvp.Value("invalid")

		s.Require().Error(err)
		s.Require().True(errors.Is(err, provider.ErrNoNamedValueInProvider))
	})

	s.Run("Value with valid name", func() {
		mvp, _ := provider.newMultiValueProvider[testStruct](newTestValueCreator("test", &testStruct{Name: "test"}))

		value, err := mvp.Value("test")

		s.Require().NotNil(value)
		s.Require().NoError(err)
		s.Require().Equal("test", value.Name)
	})

	s.Run("Value with multiple values", func() {
		mvp, _ := provider.newMultiValueProvider[testStruct](
			newTestValueCreator("test", &testStruct{Name: "test"}),
			newTestValueCreator("test2", &testStruct{Name: "test2"}),
			newTestValueCreator("test3", &testStruct{Name: "test3"}),
		)

		value, err := mvp.Value("test")
		s.Require().NotNil(value)
		s.Require().NoError(err)
		s.Require().Equal("test", value.Name)

		value2, err := mvp.Value("test2")
		s.Require().NotNil(value2)
		s.Require().NoError(err)
		s.Require().Equal("test2", value2.Name)

		value3, err := mvp.Value("test3")
		s.Require().NotNil(value3)
		s.Require().NoError(err)
		s.Require().Equal("test3", value3.Name)

		value4, err := mvp.Value("test4")
		s.Require().Nil(value4)
		s.Require().Error(err)
		s.Require().True(errors.Is(err, provider.ErrNoNamedValueInProvider))
	})
}

func newTestValueCreator[T any](name string, value *T) provider.DefaultValueCreator[T] {
	return provider.NewDefaultValueCreator(name, func() (*T, error) {
		return value, nil
	})
}
*/
