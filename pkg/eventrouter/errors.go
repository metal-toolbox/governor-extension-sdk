package eventrouter

import (
	"errors"
)

// ErrHandlerNotFound is the error returned when a handler is not found
var ErrHandlerNotFound = errors.New("handler not found")
