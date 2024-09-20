package provider_test

import (
	"sync"

	"github.com/abrosimov/go-provider"
)

// Define a test struct that embeds ChangesNotifier for testing purposes.
type TestStruct struct {
	*provider.ChangesNotifier
	value int
}

func (ts *TestStruct) SetValue(value int) error {
	ts.value = value
	return ts.NotifyListeners()
}

func (ts *TestStruct) Copy() TestStruct {
	return TestStruct{
		ChangesNotifier: ts.ChangesNotifier.Copy(),
		value:           ts.value,
	}
}

func (s *ProviderTestSuite) TestChangesNotifier() {
	testStruct := &TestStruct{
		ChangesNotifier: provider.NewChangesNotifier[TestStruct](),
		value:           42,
	}
	var wg sync.WaitGroup

	firstJobDoneCh := make(chan struct{})
	secondJobDoneCh := make(chan struct{})

	// First subscription happens before the type is provided
	ch1 := provider.SubscribeTo[TestStruct]()
	s.Require().NotNil(ch1)

	futureVals := []int{43, 44, 45}

	wg.Add(1)
	go iterateChanForSingleProvidedValue(s, &wg, ch1.GetChannel(), futureVals, firstJobDoneCh)

	_ = provider.Provide[TestStruct](func() (*TestStruct, error) {
		return testStruct, nil
	})

	ch2 := provider.SubscribeTo[TestStruct]()
	s.Require().NotNil(ch2)

	wg.Add(1)
	go iterateChanForSingleProvidedValue(s, &wg, ch2.GetChannel(), futureVals, secondJobDoneCh)

	for _, v := range futureVals {
		e := testStruct.SetValue(v)
		s.Require().NoError(e)
		<-firstJobDoneCh
		<-secondJobDoneCh
	}
	wg.Wait()
}

func iterateChanForSingleProvidedValue(
	s *ProviderTestSuite,
	wg *sync.WaitGroup,
	ch <-chan struct{},
	expectedValues []int,
	doneCh chan struct{},
) {
	defer wg.Done()
	for _, v := range expectedValues {
		<-ch
		val, err := provider.ReadOnlyValueOf[TestStruct]()
		s.Require().NoError(err)
		s.Require().Equal(v, val.value)
		doneCh <- struct{}{}
	}
}
