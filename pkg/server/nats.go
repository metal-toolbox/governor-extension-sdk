package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// NATSClient is a NATS client with some configuration
type NATSClient struct {
	conn       *nats.Conn
	logger     *zap.Logger
	prefix     string
	queueGroup string
	queueSize  int
	tracer     trace.Tracer

	subscriptions []*nats.Subscription
	messagesChan  chan *EventMessage
}

// NATSClient implements the EventClient interface
var _ EventClient = &NATSClient{}

// NATSOption is a functional configuration option for NATS
type NATSOption func(c *NATSClient)

// NewNATSClient configures and establishes a new NATS client connection
func NewNATSClient(opts ...NATSOption) (*NATSClient, error) {
	client := NATSClient{
		logger:        zap.NewNop(),
		subscriptions: []*nats.Subscription{},
		messagesChan:  make(chan *EventMessage),
	}

	for _, opt := range opts {
		opt(&client)
	}

	client.logger = client.logger.With(zap.String("component", "nats"))

	return &client, nil
}

// WithNATSConn sets the nats connection
func WithNATSConn(nc *nats.Conn) NATSOption {
	return func(c *NATSClient) {
		c.conn = nc
	}
}

// WithNATSPrefix sets the nats subscription prefix
func WithNATSPrefix(p string) NATSOption {
	return func(c *NATSClient) {
		c.prefix = p
	}
}

// WithNATSQueueGroup sets the nats subscription queue group
func WithNATSQueueGroup(q string, s int) NATSOption {
	return func(c *NATSClient) {
		c.queueGroup = q
		c.queueSize = s
	}
}

// WithNATSLogger sets the NATS client logger
func WithNATSLogger(l *zap.Logger) NATSOption {
	return func(c *NATSClient) {
		c.logger = l
	}
}

// WithNATSTracer sets the NATS client tracer
func WithNATSTracer(t trace.Tracer) NATSOption {
	return func(c *NATSClient) {
		c.tracer = t
	}
}

// Shutdown drains and closes the NATS connection
func (c *NATSClient) Shutdown() error {
	if c.conn == nil {
		return nil
	}

	c.logger.Info("shutting down NATS client")

	for _, sub := range c.subscriptions {
		c.logger.Info("unsubscribing from NATS", zap.String("subject", sub.Subject))

		if err := sub.Unsubscribe(); err != nil {
			c.logger.Warn("error unsubscribing from NATS", zap.Error(err), zap.String("subject", sub.Subject))
		}
	}

	return c.conn.Drain()
}

// Subscribe creates a subscription to the NATS subject
func (c *NATSClient) Subscribe(ctx context.Context, subject string) error {
	if c.conn == nil {
		return ErrNoNATSConnection
	}

	_, span := c.tracer.Start(ctx, "natsclient-subscribe", trace.WithAttributes(
		attribute.String("subject", subject),
	))
	defer span.End()

	handler := func(msg *nats.Msg) {
		c.logger.Info("received message", zap.String("subject", msg.Subject))

		event := &govevents.Event{}

		if err := json.Unmarshal(msg.Data, event); err != nil {
			c.logger.Error("error unmarshalling event", zap.Error(err))
			return
		}

		event.Headers = msg.Header
		msg.Subject = strings.TrimPrefix(msg.Subject, c.prefix+".")

		c.messagesChan <- &EventMessage{msg.Subject, event}
	}

	for i := 0; i < c.queueSize; i++ {
		subj := fmt.Sprintf("%s.%s", c.prefix, subject)

		subscription, err := c.conn.QueueSubscribe(subj, c.queueGroup, handler)
		if err != nil {
			return err
		}

		c.subscriptions = append(c.subscriptions, subscription)

		c.logger.Debug(
			"subscribed to NATS subject",
			zap.String("subject", subj),
			zap.Int("queue", i),
		)
	}

	return nil
}

// Messages returns a channel of messages
func (c *NATSClient) Messages() <-chan *EventMessage {
	return c.messagesChan
}
