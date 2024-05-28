package eventrouter

import (
	"context"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Middleware is a function type for defining middleware functions
type Middleware func(Handler) Handler

func (r *Router) mwInjectTraceContext(handler Handler) Handler {
	return func(ctx context.Context, event *govevents.Event) error {
		r.logger.Debug(
			"extracting trace context from event",
			zap.String("component", "trace-context-middleware"),
		)

		if event.TraceContext != nil {
			parentctx := propagation.MapCarrier(event.TraceContext)
			ctx = otel.GetTextMapPropagator().Extract(ctx, parentctx)
		}

		tracectx, span := r.tracer.Start(
			ctx, "process-event",
			trace.WithAttributes(
				attribute.String("event.erd-id", event.ExtensionResourceDefinitionID),
				attribute.String("event.extension-id", event.ExtensionID),
				attribute.String("event.resource-id", event.ExtensionResourceID),
				attribute.String("event.resource-version", event.Version),
			),
		)
		defer span.End()

		return handler(tracectx, event)
	}
}
