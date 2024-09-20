package provider

import (
	"errors"
	"fmt"
)

var (
	ErrChangeNotifierDestroyed = errors.New("change notifier has been destroyed")
)

type changeSender interface {
	send() error
}

// ChangesNotifier is a base object that can be used to notify listeners about changes.
// All you need is just to embed this struct into your object and call NotifyListeners() each time when value of base object has changed.
type ChangesNotifier struct {
	mbox   changeSender
	logger Logger
	name   string
}

// NewChangesNotifier creates a new ChangesNotifier instance which should be embedded into your structure.
func NewChangesNotifier[T any]() *ChangesNotifier {
	return &ChangesNotifier{
		name: GetTypeName[T](),
		mbox: getMailboxForType[T](),
	}
}

func NewMultiValueChangesNotifier[T any](name string) *ChangesNotifier {
	return &ChangesNotifier{
		name:   fmt.Sprintf("%s@%s", GetTypeName[T](), name),
		mbox:   getMailboxForNamedValueOfType[T](name),
		logger: logger,
	}
}

// NotifyListeners must be called each time when value of base object has been changed.
func (cn *ChangesNotifier) NotifyListeners() error {
	return cn.mbox.send()
}

func (cn *ChangesNotifier) Copy() *ChangesNotifier {
	return &ChangesNotifier{
		name:   cn.name,
		mbox:   &noopMailbox{},
		logger: cn.logger,
	}
}
