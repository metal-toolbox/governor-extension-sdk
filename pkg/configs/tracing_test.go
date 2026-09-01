package configs

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
)

// mockCollector is a minimal in-process OTLP/gRPC trace collector used to prove
// which endpoint the exporter actually dialed, without reaching a real backend.
type mockCollector struct {
	coltracepb.UnimplementedTraceServiceServer

	mu       sync.Mutex
	requests int
}

func (m *mockCollector) Export(
	_ context.Context, _ *coltracepb.ExportTraceServiceRequest,
) (*coltracepb.ExportTraceServiceResponse, error) {
	m.mu.Lock()
	m.requests++
	m.mu.Unlock()

	return &coltracepb.ExportTraceServiceResponse{}, nil
}

func (m *mockCollector) receivedRequests() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.requests
}

func startMockCollector(t *testing.T) (addr string, collector *mockCollector) {
	t.Helper()

	var lc net.ListenConfig

	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	collector = &mockCollector{}

	srv := grpc.NewServer()
	coltracepb.RegisterTraceServiceServer(srv, collector)

	go func() { _ = srv.Serve(lis) }()

	t.Cleanup(srv.Stop)

	return lis.Addr().String(), collector
}

func TestMustTracingFlags_EndpointDefaultsEmpty(t *testing.T) {
	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	MustTracingFlags(v, flags)

	// An empty default (rather than a hardcoded host:port) is what lets initTracer
	// fall back to the standard OTEL_EXPORTER_OTLP_ENDPOINT env var - see
	// TestInitTracing_FallsBackToStandardOTELEndpointEnvVar.
	assert.Empty(t, v.GetString("tracing.endpoint"))
}

func TestInitTracing_ExplicitEndpointTakesPrecedence(t *testing.T) {
	addr, collector := startMockCollector(t)

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1") // deliberately unreachable

	tr := Tracing{Enabled: true, Provider: "otlpgrpc", Endpoint: addr, Insecure: true}

	tp, err := tr.initTracer(context.Background(), "test-app")
	require.NoError(t, err)

	ctx, span := tp.Tracer("test").Start(context.Background(), "span")
	span.End()

	require.NoError(t, tp.Shutdown(ctx))

	assert.Equal(t, 1, collector.receivedRequests(),
		"explicit tracing.endpoint should be used even though OTEL_EXPORTER_OTLP_ENDPOINT points elsewhere")
}

func TestInitTracing_FallsBackToStandardOTELEndpointEnvVar(t *testing.T) {
	addr, collector := startMockCollector(t)

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+addr)

	// Endpoint deliberately left unset, mirroring the flag default from MustTracingFlags.
	tr := Tracing{Enabled: true, Provider: "otlpgrpc"}

	tp, err := tr.initTracer(context.Background(), "test-app")
	require.NoError(t, err)

	ctx, span := tp.Tracer("test").Start(context.Background(), "span")
	span.End()

	require.NoError(t, tp.Shutdown(ctx))

	assert.Equal(t, 1, collector.receivedRequests(),
		"an unset tracing.endpoint should fall back to OTEL_EXPORTER_OTLP_ENDPOINT")
}

func TestInitTracing_Disabled_ReturnsNoop(t *testing.T) {
	tr := Tracing{Enabled: false}

	tp, shutdown, err := tr.InitTracing(context.Background(), "test-app")
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NoError(t, shutdown(context.Background()))
}

func TestInitTracing_UnsupportedProvider(t *testing.T) {
	tr := Tracing{Enabled: true, Provider: "not-a-real-provider"}

	_, _, err := tr.InitTracing(context.Background(), "test-app")
	require.ErrorIs(t, err, ErrUnsupportedTracingProvider)
}
