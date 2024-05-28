package roundtripper

import (
	"net/http"

	events "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type roundtrip func(*http.Request) (*http.Response, error)

// GovExtRoundTripper is a http.RoundTripper that injects the governor
// extension contexts as required to the outgoing request headers.
type GovExtRoundTripper struct {
	logger            *zap.Logger
	roundtripperChain roundtrip
}

// RoundTrip calls the rountripper chain
func (rt *GovExtRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.roundtripperChain(req)
}

// GovExtensionRoutTripper implements the http.RoundTripper interface.
var _ http.RoundTripper = (*GovExtRoundTripper)(nil)

// NewGovExtRoundTripper returns a new GovExtRoundTripper.
func NewGovExtRoundTripper(base roundtrip, opts ...Option) *GovExtRoundTripper {
	rt := &GovExtRoundTripper{logger: zap.NewNop(), roundtripperChain: base}

	for _, opt := range opts {
		opt(rt)
	}

	rt.logger = rt.logger.With(zap.String("component", "gov-extension-roundtripper"))

	return rt
}

// Option is a function that configures a GovExtRoundTripper.
type Option func(*GovExtRoundTripper)

// WithTraceContext injects the current span context into the outgoing request
// headers.
func WithTraceContext() Option {
	return func(rt *GovExtRoundTripper) {
		next := rt.roundtripperChain
		rt.roundtripperChain = func(req *http.Request) (*http.Response, error) {
			ctx := req.Context()

			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				rt.logger.Debug(
					"injecting span context into request headers",
					zap.String("trace_id", span.SpanContext().TraceID().String()),
					zap.String("span_id", span.SpanContext().SpanID().String()),
				)

				otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
			}

			return next(req)
		}
	}
}

// WithLogger sets the logger for the GovExtRoundTripper.
func WithLogger(logger *zap.Logger) Option {
	return func(rt *GovExtRoundTripper) {
		rt.logger = logger
	}
}

// WithCorrelationID injects the current correlation ID into the outgoing
// request headers.
func WithCorrelationID() Option {
	return func(rt *GovExtRoundTripper) {
		next := rt.roundtripperChain
		rt.roundtripperChain = func(req *http.Request) (*http.Response, error) {
			ctx := req.Context()

			if cid := events.ExtractCorrelationID(ctx); cid != "" {
				rt.logger.Debug(
					"injecting correlation ID into request headers",
					zap.String("correlation_id", cid),
				)

				req.Header.Set(events.GovernorEventCorrelationIDHeader, cid)
			}

			return next(req)
		}
	}
}
