// Package eventrouter provides a router for events.
package eventrouter

import (
	"context"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// Router is the main router struct for mapping events to handlers
type Router struct {
	routes                 map[string]map[string]Handler
	mwchain                Middleware
	correlationIDProcessor *CorrelationIDProcessor

	tracer trace.Tracer
	logger *zap.Logger
}

// Option is a function that configures a Router
type Option func(*Router)

// NewRouter creates a new Router
func NewRouter(opts ...Option) *Router {
	r := &Router{
		logger: zap.NewNop(),
		tracer: noop.NewTracerProvider().Tracer("eventrouter"),
		routes: make(map[string]map[string]Handler),
		mwchain: func(handler Handler) Handler {
			return handler
		},
	}

	// apply options
	for _, opt := range opts {
		opt(r)
	}

	// set logger tags
	r.logger = r.logger.With(zap.String("component", "eventrouter"))

	return r
}

// WithLogger configures the logger for the Router
func WithLogger(logger *zap.Logger) Option {
	return func(r *Router) {
		r.logger = logger
	}
}

// WithTracer configures the tracer for the Router
func WithTracer(tracer trace.Tracer) Option {
	return func(r *Router) {
		r.tracer = tracer
		r.applyGlobalMiddleware(r.mwInjectTraceContext)
	}
}

// WithCorrelationIDProcessor configures the correlation ID processor for the Router
func WithCorrelationIDProcessor(p *CorrelationIDProcessor) Option {
	return func(r *Router) {
		r.correlationIDProcessor = p
		r.applyGlobalMiddleware(p.MWInjectCorrelationID)
	}
}

// WithMiddleware configures the middleware for the Router
func WithMiddleware(mw Middleware) Option {
	return func(r *Router) {
		r.applyGlobalMiddleware(mw)
	}
}

func (r *Router) applyGlobalMiddleware(mw Middleware) {
	next := r.mwchain
	r.mwchain = func(handler Handler) Handler {
		return mw(next(handler))
	}
}

func (r *Router) addRoute(action, subj string, handler Handler, middlewares []Middleware) {
	r.logger.Debug("adding route", zap.String("action", action), zap.String("subject", subj))

	if _, ok := r.routes[subj]; !ok {
		r.routes[subj] = make(map[string]Handler)
	}

	for _, mw := range middlewares {
		handler = mw(handler)
	}

	r.routes[subj][action] = handler
}

// Create adds a handler for the create event
func (r *Router) Create(subj string, handler Handler, middlewares ...Middleware) {
	r.addRoute(govevents.GovernorEventCreate, subj, handler, middlewares)
}

// Update adds a handler for the update event
func (r *Router) Update(subj string, handler Handler, middlewares ...Middleware) {
	r.addRoute(govevents.GovernorEventUpdate, subj, handler, middlewares)
}

// Delete adds a handler for the delete event
func (r *Router) Delete(subj string, handler Handler, middlewares ...Middleware) {
	r.addRoute(govevents.GovernorEventDelete, subj, handler, middlewares)
}

// Approve adds a handler for the approve event
func (r *Router) Approve(subj string, handler Handler, middlewares ...Middleware) {
	r.addRoute(govevents.GovernorEventApprove, subj, handler, middlewares)
}

// Deny adds a handler for the deny event
func (r *Router) Deny(subj string, handler Handler, middlewares ...Middleware) {
	r.addRoute(govevents.GovernorEventDeny, subj, handler, middlewares)
}

// Revoke adds a handler for the revoke event
func (r *Router) Revoke(subj string, handler Handler, middlewares ...Middleware) {
	r.addRoute(govevents.GovernorEventRevoke, subj, handler, middlewares)
}

// Process function finds the event handler for the event and executes it
func (r *Router) Process(ctx context.Context, subj string, event *govevents.Event) error {
	r.logger.Info(
		"processing event",
		zap.String("resource-id", event.ExtensionResourceID),
		zap.String("action", event.Action),
		zap.String("subject", subj),
	)

	if _, ok := r.routes[subj]; !ok {
		return ErrHandlerNotFound
	}

	ctx = SaveSubjectToContext(ctx, subj)

	if handler, ok := r.routes[subj][event.Action]; ok {
		return r.mwchain(handler)(ctx, event)
	}

	return nil
}

// Use adds a global middleware to the Router. This function can be used after
// the Router has been created.
func (r *Router) Use(mw Middleware) {
	r.applyGlobalMiddleware(mw)
}

// Subjects returns a list of subjects that have been registered with the
// router
func (r *Router) Subjects() []string {
	subjs := make([]string, 0, len(r.routes))

	for subj := range r.routes {
		subjs = append(subjs, subj)
	}

	return subjs
}

// Router implements EventRouter interface
var _ EventRouter = (*Router)(nil)
