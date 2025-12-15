package provider

// Test helpers for mailbox - mailbox is private, these exports are for testing only

func NewMailboxForTest(name string) *mailbox {
	return newMailbox(name)
}

func (m *mailbox) GetSubscriptionForTest() *Subscription {
	return m.GetSubscription()
}

func (m *mailbox) LenForTest() int {
	return m.Len()
}

func (m *mailbox) UnsubscribeForTest(s *Subscription) {
	m.Unsubscribe(s)
}

func SendMessageToMailbox(m *mailbox) error {
	return m.send()
}

func DestroyMailBox(m *mailbox) error {
	return m.destroy()
}
