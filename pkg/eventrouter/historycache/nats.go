package historycache

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

var (
	_ HistoryCache = (*NATSCache)(nil)
	_ configurable = (*NATSCache)(nil)
)

// NATSCache is a HistoryCache implementation that uses NATS Key-Value store.
type NATSCache struct {
	kv     nats.KeyValue
	tracer trace.Tracer
	logger *zap.Logger
}

// NewNATSCache creates a new instance of NATSCache.
func NewNATSCache(kv nats.KeyValue, opts ...Opt) *NATSCache {
	c := &NATSCache{
		kv:     kv,
		tracer: noop.NewTracerProvider().Tracer("nats-cache"),
		logger: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ExistsOrStore is an atomic operation that checks if a correlation ID exists in the cache;
// if it does not exist, it stores the ID and returns false, otherwise returns true.
// here a NATS create operation is used as a mutex, if multiple concurrent requests
// try to create the same key, only one will succeed, others will get a KeyExists error,
// and all subsequent calls will also get KeyExists error until the key is deleted.
// see https://docs.nats.io/nats-concepts/jetstream/key-value-store/kv_walkthrough#atomic-operations
func (c *NATSCache) ExistsOrStore(ctx context.Context, id string) (bool, error) {
	_, span := c.tracer.Start(ctx, "NATSCache.ExistsOrStore")
	defer span.End()

	exists := false

	if _, err := c.kv.Create(id, []byte{}); err != nil {
		if errors.Is(err, nats.ErrKeyExists) {
			exists = true
		} else {
			span.SetStatus(codes.Error, "failed to create key in NATS KV store")
			span.RecordError(err)

			return false, err
		}
	}

	span.SetAttributes(attribute.String("id", id), attribute.Bool("exists", exists))
	c.logger.Debug("exists-or-store", zap.String("id", id), zap.Bool("exists", exists))

	return exists, nil
}

// Remove removes a correlation ID from the cache.
func (c *NATSCache) Remove(ctx context.Context, id string) error {
	_, span := c.tracer.Start(ctx, "NATSCache.Remove")
	defer span.End()

	span.SetAttributes(attribute.String("id", id))

	return c.kv.Delete(id)
}

func (c *NATSCache) setLogger(l *zap.Logger) {
	c.logger = l.With(zap.String("component", "nats_cache"))
}

func (c *NATSCache) setTracer(t trace.Tracer) {
	c.tracer = t
}
