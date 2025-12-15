package historycache

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	defaultCacheSize = 128
	defaultTTL       = 1 * time.Minute
)

var (
	_ HistoryCache = (*LocalCache)(nil)
	_ configurable = (*LocalCache)(nil)
)

// LocalCache is an in-memory implementation of the HistoryCache interface.
type LocalCache struct {
	cache  *expirable.LRU[string, struct{}]
	logger *zap.Logger
	tracer trace.Tracer
	mu     *sync.Mutex
}

// NewLocalCache creates a new instance of LocalCache.
func NewLocalCache(opts ...Opt) *LocalCache {
	lc := &LocalCache{
		cache: expirable.NewLRU[string, struct{}](defaultCacheSize, nil, defaultTTL),
		mu:    &sync.Mutex{},
	}

	for _, opt := range opts {
		opt(lc)
	}

	return lc
}

// ExistsOrStore is an atomic operation that checks if a correlation ID exists in the cache;
// if it does not exist, it stores the ID and returns false, otherwise returns true.
func (lc *LocalCache) ExistsOrStore(_ context.Context, id string) (bool, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	_, exists := lc.cache.Get(id)
	if !exists {
		lc.cache.Add(id, struct{}{})
	}

	return exists, nil
}

// Remove removes a correlation ID from the cache.
func (lc *LocalCache) Remove(_ context.Context, id string) error {
	lc.mu.Lock()
	lc.cache.Remove(id)
	lc.mu.Unlock()

	return nil
}

func (lc *LocalCache) setLogger(l *zap.Logger) {
	lc.logger = l.With(zap.String("component", "local_cache"))
}

func (lc *LocalCache) setTracer(t trace.Tracer) {
	lc.tracer = t
}
