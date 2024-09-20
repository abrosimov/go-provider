package provider_test

import (
	"errors"

	"github.com/abrosimov/go-provider"
)

var (
	errFailure = errors.New("failed to create object")
)

func (s *ProviderTestSuite) TestProviderErrorEvaluation() {
	type MyObject struct {
		Name string
	}

	counter := 0
	createFn := func() (*MyObject, error) {
		counter++
		if counter >= 2 {
			return &MyObject{Name: "test"}, nil
		}
		return nil, errFailure
	}

	// Using lazy provider, so the createFn won't be evaluated until Value() is called directly.
	err := provider.Provide(createFn)
	s.Require().NoError(err)
	v1, err := provider.ValueOf[MyObject]()
	s.Require().Error(err)
	s.Require().ErrorIs(err, provider.ErrFailedToEvaluateValue)
	s.Require().Nil(v1)

	v2, err := provider.ValueOf[MyObject]()
	s.Require().NoError(err)
	s.Require().NotNil(v2)
	s.Require().Equal("test", v2.Name)
}
