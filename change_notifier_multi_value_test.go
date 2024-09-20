package provider_test

import (
	"sync"

	"github.com/abrosimov/go-provider"
)

type MultiValueTestStruct struct {
	*provider.ChangesNotifier
	value int
}

func (ts *MultiValueTestStruct) SetValue(value int) error {
	ts.value = value
	return ts.NotifyListeners()
}

func (ts *MultiValueTestStruct) Copy() MultiValueTestStruct {
	return MultiValueTestStruct{
		ChangesNotifier: ts.ChangesNotifier.Copy(),
		value:           ts.value,
	}
}

//nolint:funlen // it's ok here
func (s *ProviderTestSuite) TestMultiValueChangeNotifier() {
	phpStruct := &MultiValueTestStruct{
		ChangesNotifier: provider.NewMultiValueChangesNotifier[MultiValueTestStruct]("php"),
		value:           0,
	}
	golangStruct := &MultiValueTestStruct{
		ChangesNotifier: provider.NewMultiValueChangesNotifier[MultiValueTestStruct]("golang"),
		value:           0,
	}

	futurePhpVars := []int{5, 4, 3, 2, 1}
	futureGolangVars := []int{1, 2, 3, 4, 5}

	doneChannels := newDoneChannelsStore()

	phpChBeforeProvided := provider.SubscribeToNamedValueOf[MultiValueTestStruct]("php")
	s.Require().NotNil(phpChBeforeProvided)

	golangChBeforeProvided := provider.SubscribeToNamedValueOf[MultiValueTestStruct]("golang")
	s.Require().NotNil(golangChBeforeProvided)

	var wg sync.WaitGroup

	const totalWGSize = 4
	wg.Add(totalWGSize)
	go iterateChanForMultiProvidedValue(s, "php", &wg, phpChBeforeProvided, futurePhpVars, doneChannels.phpBeforeProvidedCh)
	go iterateChanForMultiProvidedValue(s, "golang", &wg, golangChBeforeProvided, futureGolangVars, doneChannels.goBeforeProvidedCh)

	err := provider.ProvideMultiValue[MultiValueTestStruct](
		provider.NewDefaultValueCreator(
			"golang",
			func() (*MultiValueTestStruct, error) {
				return golangStruct, nil
			},
		),
		provider.NewDefaultValueCreator(
			"php",
			func() (*MultiValueTestStruct, error) {
				return phpStruct, nil
			},
		),
	)
	s.Require().NoError(err)

	phpChAfterProvided := provider.SubscribeToNamedValueOf[MultiValueTestStruct]("php")
	s.Require().NotNil(phpChAfterProvided)

	golangChAfterProvided := provider.SubscribeToNamedValueOf[MultiValueTestStruct]("golang")
	s.Require().NotNil(golangChAfterProvided)

	go iterateChanForMultiProvidedValue(s, "php", &wg, phpChAfterProvided, futurePhpVars, doneChannels.phpAfterProvidedCh)
	go iterateChanForMultiProvidedValue(s, "golang", &wg, golangChAfterProvided, futureGolangVars, doneChannels.goAfterProvidedCh)

	iterateOverFutureVars(s, langNamePHP, phpStruct, futurePhpVars, doneChannels)
	iterateOverFutureVars(s, langNameGolang, golangStruct, futureGolangVars, doneChannels)

	wg.Wait()
}

//nolint:lll // ok here
func iterateOverFutureVars(s *ProviderTestSuite, lang langName, providedValue *MultiValueTestStruct, futureVars []int, doneChannels doneChannelsStore) {
	for _, v := range futureVars {
		err := providedValue.SetValue(v)
		s.Require().NoError(err)

		doneChannels.ensureThatC(lang)
	}
}

func iterateChanForMultiProvidedValue(
	s *ProviderTestSuite,
	name string,
	wg *sync.WaitGroup,
	subscription *provider.Subscription,
	expectedValues []int,
	doneCh chan struct{},
) {
	defer wg.Done()
	for _, v := range expectedValues {
		<-subscription.GetChannel()
		val, err := provider.ReadOnlyMultiValueOf[MultiValueTestStruct](name)
		s.Require().NoError(err)
		s.Require().Equal(v, val.value)
		doneCh <- struct{}{}
	}
}
