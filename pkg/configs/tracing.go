package configs

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Tracing holds tracing configuration
type Tracing struct {
	Enabled     bool   `mapstructure:"enabled"`
	Provider    string `mapstructure:"provider"`
	Endpoint    string `mapstructure:"endpoint"`
	Environment string `mapstructure:"environment"`
	Insecure    bool   `mapstructure:"insecure"`
}

// MustTracingFlags registers tracing related flags and binds them to viper
// Panics on error
func MustTracingFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.Bool("tracing", false, "enable tracing support")
	viperBindFlag(v, "tracing.enabled", flags.Lookup("tracing"))
	flags.String("tracing-provider", "otlpgrpc", "tracing provider to use")
	viperBindFlag(v, "tracing.provider", flags.Lookup("tracing-provider"))
	flags.String("tracing-endpoint", "trace:4317", "endpoint where traces are sent")
	viperBindFlag(v, "tracing.endpoint", flags.Lookup("tracing-endpoint"))
	flags.String("tracing-environment", "production", "environment value in traces")
	viperBindFlag(v, "tracing.environment", flags.Lookup("tracing-environment"))
	flags.Bool("tracing-insecure", false, "use insecure connection for tracing endpoint")
	viperBindFlag(v, "tracing.insecure", flags.Lookup("tracing-insecure"))
}

// TPShutdown defines a function to shutdown a TracerProvider.
type TPShutdown func(context.Context) error

// InitTracing initializes tracing based on the application configuration.
func (t Tracing) InitTracing(ctx context.Context, appName string) (trace.TracerProvider, TPShutdown, error) {
	if t.Enabled {
		tp, err := t.initTracer(ctx, appName)
		if err != nil {
			return nil, nil, err
		}

		return tp, tp.Shutdown, nil
	}

	return noop.NewTracerProvider(), func(_ context.Context) error { return nil }, nil
}

// initTracer returns an OpenTelemetry TracerProvider.
func (t Tracing) initTracer(_ context.Context, appName string) (*tracesdk.TracerProvider, error) {
	var (
		client otlptrace.Client
		err    error
	)

	switch t.Provider {
	case "otlpgrpc":
		clientOptions := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(t.Endpoint)}
		if t.Insecure {
			clientOptions = append(clientOptions, otlptracegrpc.WithInsecure())
		}

		client = otlptracegrpc.NewClient(clientOptions...)
	default:
		return nil, fmt.Errorf("%w: provider: %s", ErrUnsupportedTracingProvider, t.Provider)
	}

	exp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(appName),
			attribute.String("environment", t.Environment),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}
