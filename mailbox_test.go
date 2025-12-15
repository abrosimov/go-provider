package provider_test

import (
	"time"

	"github.com/abrosimov/go-provider"
)

func (s *ProviderTestSuite) TestMailBox() {
	mbox := provider.NewMailboxForTest("test")

	sub1 := mbox.GetSubscriptionForTest()
	s.Require().Equal(1, mbox.LenForTest())
	err := provider.SendMessageToMailbox(mbox)
	s.Require().NoError(err)

	assertChannelHasIncomingMessage(s, sub1.GetChannel())

	sub2 := mbox.GetSubscriptionForTest()
	s.Require().Equal(2, mbox.LenForTest())
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

	s.Require().Equal(0, mbox.LenForTest())
	_, isOpened := <-sub1.GetChannel()
	s.Require().False(isOpened)
	_, isOpened = <-sub2.GetChannel()
	s.Require().False(isOpened)
}

func (s *ProviderTestSuite) TestSubscribeUnsubscribe() {
	mbox := provider.NewMailboxForTest("test")

	sub1 := mbox.GetSubscriptionForTest()
	s.Require().Equal(1, mbox.LenForTest())
	mbox.UnsubscribeForTest(sub1)
	s.Require().Equal(0, mbox.LenForTest())
	_, isOpened := <-sub1.GetChannel()
	s.Require().False(isOpened)

	sub1 = mbox.GetSubscriptionForTest()
	s.Require().Equal(1, mbox.LenForTest())
	sub2 := mbox.GetSubscriptionForTest()
	s.Require().Equal(2, mbox.LenForTest())
	sub3 := mbox.GetSubscriptionForTest()
	s.Require().Equal(3, mbox.LenForTest())

	err := provider.SendMessageToMailbox(mbox)
	s.Require().NoError(err)

	assertChannelHasIncomingMessage(s, sub1.GetChannel())
	assertChannelHasIncomingMessage(s, sub2.GetChannel())
	assertChannelHasIncomingMessage(s, sub3.GetChannel())

	mbox.UnsubscribeForTest(sub2)
	s.Require().Equal(2, mbox.LenForTest())
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
