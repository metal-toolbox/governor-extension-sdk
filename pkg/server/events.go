package server

import (
	"context"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// EventClient is an interface for the event client
type EventClient interface {
	Subscribe(ctx context.Context, subject string) error
	Messages() <-chan *EventMessage
	Shutdown() error
}

// EventMessage is a wrapper for a governor event message
type EventMessage struct {
	Subject string
	Event   *govevents.Event
}

// Subscribe subscribes to all subjects related to the extension
func (s *Server) Subscribe(ctx context.Context) error {
	s.logger.Info("subscribing to event subjects")

	ctx, span := s.tracer.Start(ctx, "subscribe")
	defer span.End()

	// subscribe to extension events
	for _, subj := range s.eventRouter.Subjects() {
		s.logger.Info("subscribing to subject", zap.String("subject", subj))

		if err := s.eventClient.Subscribe(ctx, subj); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return err
		}
	}

	return nil
}

// ListenEvents listens for events from the governor api
func (s *Server) ListenEvents(ctx context.Context) {
	s.logger.Info("starting event listeners")

	for {
		select {
		case msg := <-s.eventClient.Messages():
			s.logger.Info("received governor event")

			go func(ctx context.Context) {
				if err := s.eventRouter.Process(ctx, msg.Subject, msg.Event); err != nil {
					s.logger.Error("error processing event", zap.Error(err))
				}
			}(ctx)

		case <-ctx.Done():
			s.logger.Info("context cancelled, shutting down")
			return
		}
	}
}
