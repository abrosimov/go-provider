package provider

var NewMailBox = NewMailbox

func SendMessageToMailbox(m *Mailbox) error {
	return m.send()
}

func DestroyMailBox(m *Mailbox) error {
	return m.destroy()
}
