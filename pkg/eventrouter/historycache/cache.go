// Package historycache provides a cache for storing and retrieving historical
// correlation IDs to help in determining whether certain events can be skipped
// during processing based on predefined skip strategies.
package historycache

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// HistoryCache defines the interface for a cache that stores historical correlation IDs.
type HistoryCache interface {
	// ExistsOrStore is an atomic operation that checks if a correlation ID exists in the cache;
	// if it does not exist, it stores the ID and returns false, otherwise returns true.
	ExistsOrStore(ctx context.Context, id string) (bool, error)
	// Remove removes a correlation ID from the cache.
	Remove(ctx context.Context, id string) error
}

type configurable interface {
	setLogger(l *zap.Logger)
	setTracer(t trace.Tracer)
}

// Opt is a functional option for configuring a HistoryCache implementation.
type Opt func(c configurable)

// WithLogger is an option to set a custom logger for the HistoryCache implementation.
func WithLogger(l *zap.Logger) Opt {
	return func(c configurable) {
		c.setLogger(l)
	}
}

// WithTracer is an option to set a custom tracer for the HistoryCache implementation.
func WithTracer(t trace.Tracer) Opt {
	return func(c configurable) {
		c.setTracer(t)
	}
}
