package provider_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/abrosimov/go-provider"
)

type ProviderTestSuite struct {
	suite.Suite
}

func (s *ProviderTestSuite) SetupTest() {
	err := provider.ResetRegistry()
	s.Require().NoError(err)
}

func TestProviderTestSuite(t *testing.T) {
	// goleak.VerifyTestMain() // TODO: Enable it for tests
	testSuite := new(ProviderTestSuite)
	suite.Run(t, testSuite)
}
