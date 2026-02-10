package bus

import "errors"

var (
	ErrTimeout       = errors.New("message bus timeout")
	ErrHandlerNotFound = errors.New("handler not found")
	ErrClosed        = errors.New("message bus closed")
)