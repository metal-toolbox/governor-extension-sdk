package configs

import "errors"

// ErrUnsupportedTracingProvider is returned when an unsupported tracing provider is specified.
var ErrUnsupportedTracingProvider = errors.New("unsupported tracing provider")
