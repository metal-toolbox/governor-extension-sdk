package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	governor "github.com/metal-toolbox/governor-api/pkg/client"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventprocessor"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Status is an enum type for the server status
type Status string

const (
	// StatusUp is the status when the server is up
	StatusUp Status = "UP"
	// StatusDisabled is the status when the extension is disabled
	StatusDisabled Status = "DISABLED"
	// StatusBootstrapping is the status when the server is bootstrapping
	StatusBootstrapping Status = "BOOTSTRAPPING"
)

// Server implements the HTTP Server
type Server struct {
	Listen          string
	Debug           bool
	AuditFileWriter io.Writer

	erdDir         string
	logger         *zap.Logger
	extensionID    string
	extension      *v1alpha1.Extension
	governorClient *governor.Client
	eventClient    EventClient
	status         Status
	tracer         trace.Tracer

	eventRouter eventrouter.EventRouter
	processors  []eventprocessor.EventProcessor
}

// Option is a function that configures a Server
type Option func(*Server)

// NewServer creates a new HTTP server
func NewServer(
	listen, extensionID, erdDir string,
	opts ...Option,
) *Server {
	s := &Server{
		Listen:          listen,
		Debug:           false,
		AuditFileWriter: os.Stdout,

		logger:      zap.NewNop(),
		extensionID: extensionID,
		erdDir:      erdDir,

		processors: []eventprocessor.EventProcessor{},
	}

	for _, opt := range opts {
		opt(s)
	}

	s.logger = s.logger.With(zap.String("component", "server"))

	if s.eventRouter == nil {
		s.eventRouter = eventrouter.NewRouter(
			eventrouter.WithLogger(s.logger),
			eventrouter.WithTracer(s.tracer),
			eventrouter.WithCorrelationIDProcessor(eventrouter.NewCorrelationIDProcessor(
				eventrouter.CorrelationIDProcessorWithLogger(s.logger),
				eventrouter.CorrelationIDProcessorWithSkipStrategyUpdateOnly(),
			)),
		)
	}

	return s
}

// WithEventProcessor adds an event processor to the server
func WithEventProcessor(p eventprocessor.EventProcessor) Option {
	return func(s *Server) {
		s.processors = append(s.processors, p)
	}
}

// WithEventRouter sets the event router for the server
func WithEventRouter(er eventrouter.EventRouter) Option {
	return func(s *Server) {
		s.eventRouter = er
	}
}

// WithLogger sets the logger for the server
func WithLogger(logger *zap.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithDebug sets the debug flag for the server
func WithDebug(dbg bool) Option {
	return func(s *Server) {
		s.Debug = dbg
	}
}

// WithAuditFileWriter sets the audit file writer for the server
func WithAuditFileWriter(w io.Writer) Option {
	return func(s *Server) {
		s.AuditFileWriter = w
	}
}

// WithGovernorClient sets the governor client for the server
func WithGovernorClient(c *governor.Client) Option {
	return func(s *Server) {
		s.governorClient = c
	}
}

// WithNATSClient sets the nats client for the server
func WithNATSClient(c *NATSClient) Option {
	return func(s *Server) {
		s.eventClient = c
	}
}

// WithTracer sets the tracer for the server
func WithTracer(t trace.Tracer) Option {
	return func(s *Server) {
		s.tracer = t
	}
}

var (
	readTimeout     = 10 * time.Second
	writeTimeout    = 20 * time.Second
	corsMaxAge      = 12 * time.Hour
	shutdownTimeout = 5 * time.Second
)

func (s *Server) setup() *gin.Engine {
	// Setup default gin router
	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
		MaxAge:           corsMaxAge,
	}))

	p := ginprometheus.NewPrometheus("gin")

	// Remove any params from the URL string to keep the number of labels down
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}

	p.Use(r)

	customLogger := s.logger.With(zap.String("component", "httpsrv"))
	r.Use(
		ginzap.GinzapWithConfig(customLogger, &ginzap.Config{
			TimeFormat: time.RFC3339,
			SkipPaths:  []string{"/healthz", "/healthz/readiness", "/healthz/liveness"},
			UTC:        true,
		}),
	)

	r.Use(ginzap.RecoveryWithZap(s.logger.With(zap.String("component", "httpsrv")), true))

	tp := otel.GetTracerProvider()
	if tp != nil {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		r.Use(otelgin.Middleware(hostname, otelgin.WithTracerProvider(tp)))
	}

	// Health endpoints
	r.GET("/healthz", s.livenessCheck)
	r.GET("/healthz/liveness", s.livenessCheck)
	r.GET("/healthz/readiness", s.readinessCheck)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "invalid request - route not found"})
	})

	return r
}

func (s *Server) newHTTPServer() *http.Server {
	if !s.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return &http.Server{
		Handler:      s.setup(),
		Addr:         s.Listen,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

// Run will start the server listening on the specified address
func (s *Server) Run(ctx context.Context) error {
	httpsrv := s.newHTTPServer()

	go func() {
		if err := httpsrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	startupCtx, span := s.tracer.Start(ctx, "server-startup")
	defer span.End()

	if err := s.Bootstrap(startupCtx); err != nil {
		s.logger.Error("failed bootstrapping extension", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		return err
	}

	if err := s.Subscribe(startupCtx); err != nil {
		s.logger.Error("failed subscribing to extension events", zap.Error(err))
	}

	go s.ListenEvents(ctx)
	span.End()

	// wait foir shutdown
	<-ctx.Done()
	s.logger.Info("context cancelled, shutting down")

	shutdownctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := httpsrv.Shutdown(shutdownctx); err != nil {
		return err
	}

	if err := s.eventClient.Shutdown(); err != nil {
		return err
	}

	s.logger.Info("server shutdown cleanly", zap.String("time", time.Now().UTC().Format(time.RFC3339)))

	return nil
}

// livenessCheck ensures that the server is up and responding
func (s *Server) livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

// readinessCheck ensures that the server is up and that we are able to process requests.
func (s *Server) readinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": s.status,
	})
}
