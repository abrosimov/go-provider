package provider_test

import (
	"sync"

	"github.com/abrosimov/go-provider"
)

type testGenericConstraint interface {
	whoAmI() langName
}

type langPHP struct{}
type langGolang struct{}

func (l langPHP) whoAmI() langName {
	return langNamePHP
}

func (l langGolang) whoAmI() langName {
	return langNameGolang
}

// Define a generic test struct that embeds ChangesNotifier for testing purposes.
type TestGenericStruct[T testGenericConstraint] struct {
	*provider.ChangesNotifier
	value int
}

func (ts *TestGenericStruct[T]) Copy() TestGenericStruct[T] {
	return TestGenericStruct[T]{
		ChangesNotifier: ts.ChangesNotifier.Copy(),
		value:           ts.value,
	}
}

func (ts *TestGenericStruct[T]) SetValue(value int) error {
	ts.value = value
	return ts.NotifyListeners()
}

//nolint:funlen // it's ok here
func (s *ProviderTestSuite) TestChangeNotifierOfGenericType() {
	phpStruct := &TestGenericStruct[langPHP]{
		ChangesNotifier: provider.NewChangesNotifier[TestGenericStruct[langPHP]](),
		value:           0,
	}
	golangStruct := &TestGenericStruct[langGolang]{
		ChangesNotifier: provider.NewChangesNotifier[TestGenericStruct[langGolang]](),
		value:           5,
	}

	futurePhpVals := []int{5, 4, 3, 2, 1}
	futureGolangVals := []int{1, 2, 3, 4, 5}

	doneChannels := newDoneChannelsStore()

	phpSubBeforeProvided := provider.SubscribeTo[TestGenericStruct[langPHP]]()
	s.Require().NotNil(phpSubBeforeProvided)

	goSubBeforeProvided := provider.SubscribeTo[TestGenericStruct[langGolang]]()
	s.Require().NotNil(goSubBeforeProvided)

	var wg sync.WaitGroup

	const totalWGSize = 4
	wg.Add(totalWGSize)
	go iterateChanForProvidedValueOfGeneric(s, readPhpValue, &wg, phpSubBeforeProvided, futurePhpVals, doneChannels.phpBeforeProvidedCh)
	go iterateChanForProvidedValueOfGeneric(s, readGolangValue, &wg, goSubBeforeProvided, futureGolangVals, doneChannels.goBeforeProvidedCh)

	err := provider.Provide[TestGenericStruct[langPHP]](
		func() (*TestGenericStruct[langPHP], error) {
			return phpStruct, nil
		},
	)
	s.Require().NoError(err)
	err = provider.Provide[TestGenericStruct[langGolang]](
		func() (*TestGenericStruct[langGolang], error) {
			return golangStruct, nil
		},
	)
	provider.GetProvidedTypes()
	s.Require().NoError(err)

	phpSubAfterProvided := provider.SubscribeTo[TestGenericStruct[langPHP]]()
	s.Require().NotNil(phpSubAfterProvided)

	goSubAfterProvided := provider.SubscribeTo[TestGenericStruct[langGolang]]()
	s.Require().NotNil(goSubAfterProvided)

	go iterateChanForProvidedValueOfGeneric(s, readPhpValue, &wg, phpSubAfterProvided, futurePhpVals, doneChannels.phpAfterProvidedCh)
	go iterateChanForProvidedValueOfGeneric(s, readGolangValue, &wg, goSubAfterProvided, futureGolangVals, doneChannels.goAfterProvidedCh)

	iterateOverFutureVarsOfGeneric(s, phpStruct, futurePhpVals, doneChannels.getDoneChannelsFor(langNamePHP))
	iterateOverFutureVarsOfGeneric(s, golangStruct, futureGolangVals, doneChannels.getDoneChannelsFor(langNameGolang))

	wg.Wait()
}

func iterateChanForProvidedValueOfGeneric(
	s *ProviderTestSuite,
	valueGetter func() (int, error),
	wg *sync.WaitGroup,
	subscription *provider.Subscription,
	expectedValues []int,
	doneCh chan struct{},
) {
	defer wg.Done()
	for _, v := range expectedValues {
		<-subscription.GetChannel()
		val, err := valueGetter()
		s.Require().NoError(err)
		s.Require().Equal(v, val)
		doneCh <- struct{}{}
	}
}

type valueSetter interface {
	SetValue(int) error
}

func iterateOverFutureVarsOfGeneric(s *ProviderTestSuite, providedValue valueSetter, futureVars []int, doneChannels []chan struct{}) {
	for _, v := range futureVars {
		err := providedValue.SetValue(v)
		s.Require().NoError(err)

		var wg sync.WaitGroup
		wg.Add(len(doneChannels))
		for _, ch := range doneChannels {
			go func(ch <-chan struct{}) {
				<-ch
				wg.Done()
			}(ch)
		}
		wg.Wait()
	}
}

func readPhpValue() (int, error) {
	v, err := provider.ReadOnlyValueOf[TestGenericStruct[langPHP]]()
	if err != nil {
		return -1, err
	}

	return v.value, nil
}

func readGolangValue() (int, error) {
	v, err := provider.ReadOnlyValueOf[TestGenericStruct[langGolang]]()
	if err != nil {
		return -1, err
	}

	return v.value, nil
}
