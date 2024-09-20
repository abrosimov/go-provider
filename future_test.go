package provider_test

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/abrosimov/go-provider"
)

func (s *ProviderTestSuite) TestWait() {
	f := provider.FutureOf[MyGlobalObject](zerolog.Nop())
	runProvideWithDelay(50 * time.Millisecond)
	f.Wait()
	v, err := f.Get(context.Background())
	s.Require().NoError(err)
	s.Require().Equal("test", v.name)
}

func (s *ProviderTestSuite) TestWaitForFailsWithTimeout() {
	f := provider.FutureOf[MyGlobalObject](zerolog.Nop())
	runProvideWithDelay(100 * time.Millisecond)
	err := f.WaitFor(time.Millisecond * 10)
	s.Require().Error(err)
}

func (s *ProviderTestSuite) TestWaitForSucceed() {
	f := provider.FutureOf[MyGlobalObject](zerolog.Nop())
	runProvideWithDelay(30 * time.Millisecond)
	err := f.WaitFor(time.Millisecond * 60)
	s.Require().NoError(err)
	v, err := f.Get(context.Background())
	s.Require().NoError(err)
	s.Require().Equal("test", v.name)
}

func (s *ProviderTestSuite) TestGetWithCtxCanceled() {
	f := provider.FutureOf[MyGlobalObject](zerolog.Nop())
	runProvideWithDelay(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.Get(ctx)
	s.Require().Error(err)
}

func (s *ProviderTestSuite) TestGetWith() {
	f := provider.FutureOf[MyGlobalObject](zerolog.Nop())
	runProvideWithDelay(0 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := f.Get(ctx)
	s.Require().NoError(err)
	v, err := f.Get(context.Background())
	s.Require().NoError(err)
	s.Require().Equal("test", v.name)
}

func runProvideWithDelay(delay time.Duration) {
	go func() {
		<-time.After(delay)
		_ = provider.Provide[MyGlobalObject](func() (*MyGlobalObject, error) {
			return &MyGlobalObject{name: "test"}, nil
		})
	}()
}
