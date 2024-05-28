package server

import "errors"

// ErrNoNATSConnection is returned when there is no NATS connection
var ErrNoNATSConnection = errors.New("no NATS connection")
