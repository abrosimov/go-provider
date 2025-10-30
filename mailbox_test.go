package provider_test

import (
	"time"

	"github.com/abrosimov/go-provider"
)

func (s *ProviderTestSuite) TestMailBox() {
	mbox := provider.NewMailBox("test")

	sub1 := mbox.GetSubscription()
	s.Require().Equal(1, mbox.Len())
	err := provider.SendMessageToMailbox(mbox)
	s.Require().NoError(err)

	assertChannelHasIncomingMessage(s, sub1.GetChannel())

	sub2 := mbox.GetSubscription()
	s.Require().Equal(2, mbox.Len())
	err = provider.SendMessageToMailbox(mbox)
	s.Require().NoError(err)

	assertChannelHasIncomingMessage(s, sub1.GetChannel())
	// sub2 was created after mailbox received its first change, so it'll have one more message because we want
	// to notify new subscriber about changed state of the object.
	assertChannelHasIncomingMessage(s, sub2.GetChannel())
	assertChannelHasIncomingMessage(s, sub2.GetChannel())
	s.Require().Empty(sub1.GetChannel())
	s.Require().Empty(sub2.GetChannel())

	err = provider.DestroyMailBox(mbox)
	s.Require().NoError(err)

	s.Require().Equal(0, mbox.Len())
	_, isOpened := <-sub1.GetChannel()
	s.Require().False(isOpened)
	_, isOpened = <-sub2.GetChannel()
	s.Require().False(isOpened)
}

func (s *ProviderTestSuite) TestSubscribeUnsubscribe() {
	mbox := provider.NewMailBox("test")

	sub1 := mbox.GetSubscription()
	s.Require().Equal(1, mbox.Len())
	mbox.Unsubscribe(sub1)
	s.Require().Equal(0, mbox.Len())
	_, isOpened := <-sub1.GetChannel()
	s.Require().False(isOpened)

	sub1 = mbox.GetSubscription()
	s.Require().Equal(1, mbox.Len())
	sub2 := mbox.GetSubscription()
	s.Require().Equal(2, mbox.Len())
	sub3 := mbox.GetSubscription()
	s.Require().Equal(3, mbox.Len())

	err := provider.SendMessageToMailbox(mbox)
	s.Require().NoError(err)

	assertChannelHasIncomingMessage(s, sub1.GetChannel())
	assertChannelHasIncomingMessage(s, sub2.GetChannel())
	assertChannelHasIncomingMessage(s, sub3.GetChannel())

	mbox.Unsubscribe(sub2)
	s.Require().Equal(2, mbox.Len())
	_, isOpened = <-sub2.GetChannel()
	s.Require().False(isOpened)
}

func assertChannelHasIncomingMessage(s *ProviderTestSuite, ch <-chan struct{}) {
	timer := time.NewTimer(100 * time.Millisecond)
	select {
	case <-ch:
	case <-timer.C:
		s.Fail("channel should have received a message")
	}
}
