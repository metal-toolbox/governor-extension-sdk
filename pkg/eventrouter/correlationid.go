package eventrouter

import (
	"context"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter/historycache"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// CorrelationIDProcessor is responsible for processing events based on correlation ID and skip strategy.
type CorrelationIDProcessor struct {
	logger *zap.Logger

	// histcache is a local cache of update history, specifically records the
	// correlation ID. this is to prevent the extension from reacting to its own
	// updates
	histcache historycache.HistoryCache
	// skippableRoutes is a map of routes that can be skipped based on the
	// correlation ID and skip strategy
	skippableRoutes map[string]map[string]struct{}
}

// CorrelationIDProcessorOpt is a function type for configuring CorrelationIDProcessor.
type CorrelationIDProcessorOpt func(*CorrelationIDProcessor)

// NewCorrelationIDProcessor creates a new instance of CorrelationIDProcessor with the provided options.
func NewCorrelationIDProcessor(opts ...CorrelationIDProcessorOpt) *CorrelationIDProcessor {
	p := &CorrelationIDProcessor{
		logger:          zap.NewNop(),
		histcache:       historycache.NewLocalCache(),
		skippableRoutes: make(map[string]map[string]struct{}),
	}

	// default skip strategy is to skip only update events
	CorrelationIDProcessorWithSkipStrategyUpdateOnly()(p)

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// CorrelationIDProcessorWithHistoryCache sets the history cache for CorrelationIDProcessor.
func CorrelationIDProcessorWithHistoryCache(histcache historycache.HistoryCache) CorrelationIDProcessorOpt {
	return func(p *CorrelationIDProcessor) {
		p.histcache = histcache
	}
}

// CorrelationIDProcessorWithLogger sets the logger for CorrelationIDProcessor.
func CorrelationIDProcessorWithLogger(logger *zap.Logger) CorrelationIDProcessorOpt {
	return func(p *CorrelationIDProcessor) {
		p.logger = logger
	}
}

// CorrelationIDProcessorWithSkipStrategyUpdateOnly sets the skip strategy to skip only update events.
func CorrelationIDProcessorWithSkipStrategyUpdateOnly() CorrelationIDProcessorOpt {
	return func(p *CorrelationIDProcessor) {
		p.skippableRoutes = map[string]map[string]struct{}{
			govevents.GovernorEventUpdate: {"*": {}},
		}
	}
}

// CorrelationIDProcessorWithSkipStrategySkipAll sets the skip strategy to skip all events.
func CorrelationIDProcessorWithSkipStrategySkipAll() CorrelationIDProcessorOpt {
	return func(p *CorrelationIDProcessor) {
		p.skippableRoutes = map[string]map[string]struct{}{
			govevents.GovernorEventUpdate:  {"*": {}},
			govevents.GovernorEventCreate:  {"*": {}},
			govevents.GovernorEventDelete:  {"*": {}},
			govevents.GovernorEventApprove: {"*": {}},
			govevents.GovernorEventDeny:    {"*": {}},
			govevents.GovernorEventRevoke:  {"*": {}},
		}
	}
}

// CorrelationIDProcessorWithSkipStrategyCustom sets a custom skip strategy for CorrelationIDProcessor.
func CorrelationIDProcessorWithSkipStrategyCustom(sr map[string]map[string]struct{}) CorrelationIDProcessorOpt {
	return func(p *CorrelationIDProcessor) {
		p.skippableRoutes = sr
	}
}

// ShouldSkip returns true if the event should be skipped based on the
// correlation ID and the skip strategy.
//
// A process is only skipped when the correlation ID is not empty and the
// correlation ID is found in the history cache. The skip strategy is applied
// to determine if the event should be skipped.
func (p *CorrelationIDProcessor) ShouldSkip(ctx context.Context, cid, action, subj string) (bool, error) {
	exists, err := p.histcache.ExistsOrStore(ctx, cid)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	if _, ok := p.skippableRoutes[action]; ok {
		if _, ok := p.skippableRoutes[action]["*"]; ok {
			return true, nil
		}

		if _, ok := p.skippableRoutes[action][subj]; ok {
			return true, nil
		}
	}

	return false, nil
}

// MWInjectCorrelationID returns a middleware that injects the correlation ID into the context.
func (p *CorrelationIDProcessor) MWInjectCorrelationID(next Handler) Handler {
	return func(ctx context.Context, event *govevents.Event) error {
		var (
			headers nats.Header = event.Headers
			cid     string
		)

		if headers != nil {
			cid = headers.Get(govevents.GovernorEventCorrelationIDHeader)

			p.logger.Debug(
				"extracted correlation ID from event",
				zap.String("correlation-id", cid),
				zap.String("component", "correlation-id-middleware"),
			)
		}

		subj := GetSubjectFromContext(ctx)

		skip, err := p.ShouldSkip(ctx, cid, event.Action, subj)
		if err != nil {
			return err
		}

		if subj != "" && cid != "" && skip {
			p.logger.Info(
				"skipping event",
				zap.String("action", event.Action),
				zap.String("subject", subj),
				zap.String("resource-id", event.ExtensionResourceID),
				zap.String("correlation-id", cid),
				zap.String("component", "correlation-id-middleware"),
			)

			return nil
		}

		nextctx := govevents.InjectCorrelationID(ctx, cid)
		err = next(nextctx, event)

		return err
	}
}
