package provider_test

import "sync"

type langName string

const (
	langNamePHP    langName = "php"
	langNameGolang langName = "golang"
)

type doneChannelsStore struct {
	phpAfterProvidedCh  chan struct{}
	phpBeforeProvidedCh chan struct{}
	goAfterProvidedCh   chan struct{}
	goBeforeProvidedCh  chan struct{}
}

func newDoneChannelsStore() doneChannelsStore {
	return doneChannelsStore{
		phpAfterProvidedCh:  make(chan struct{}),
		phpBeforeProvidedCh: make(chan struct{}),
		goAfterProvidedCh:   make(chan struct{}),
		goBeforeProvidedCh:  make(chan struct{}),
	}
}

func (d doneChannelsStore) getDoneChannelsFor(lang langName) []chan struct{} {
	if lang == langNamePHP {
		return []chan struct{}{d.phpBeforeProvidedCh, d.phpAfterProvidedCh}
	}

	return []chan struct{}{d.goBeforeProvidedCh, d.goAfterProvidedCh}
}

func (d doneChannelsStore) ensureThatC(lang langName) {
	var wg sync.WaitGroup
	channels := d.getDoneChannelsFor(lang)
	wg.Add(len(channels))
	for _, ch := range channels {
		go func(ch <-chan struct{}) {
			<-ch
			wg.Done()
		}(ch)
	}

	wg.Wait()
}
