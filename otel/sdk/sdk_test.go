package otelsdk

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/platform-smith-labs/japi-core/v3/otel"
)

// TraceID is the otel package's helper; aliased for brevity.
var TraceID = otel.TraceID

// keepingExporter wraps the in-memory exporter so that Shutdown does not
// discard the spans it collected (InMemoryExporter.Shutdown resets them).
type keepingExporter struct {
	*tracetest.InMemoryExporter
	mu       sync.Mutex
	shutdown bool
}

func (e *keepingExporter) Shutdown(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shutdown = true
	return nil
}

// clearOTelEnv blanks variables that would otherwise let the host
// environment change resource or batching behaviour under test.
func clearOTelEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"OTEL_SERVICE_NAME", "OTEL_RESOURCE_ATTRIBUTES"} {
		t.Setenv(k, "")
	}
}

func setup(t *testing.T, cfg Config) func(context.Context) error {
	t.Helper()
	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	return shutdown
}

func TestSetup_ExportsSpanAndShutdownFlushes(t *testing.T) {
	clearOTelEnv(t)
	exp := &keepingExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
	shutdown := setup(t, Config{ServiceName: "fallback-name", SpanExporter: exp})

	ctx, span := otelapi.Tracer("test").Start(context.Background(), "op")
	traceID := TraceID(ctx)
	span.End()

	// The batch processor holds the span until a flush; nothing yet.
	if n := len(exp.GetSpans()); n != 0 {
		t.Fatalf("spans exported before shutdown = %d, want 0", n)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans exported after shutdown = %d, want 1", len(spans))
	}
	if got := spans[0].SpanContext.TraceID().String(); got != traceID {
		t.Errorf("exported trace id = %s, TraceID(ctx) = %s", got, traceID)
	}
	if !exp.shutdown {
		t.Error("exporter was not shut down")
	}
	if got := serviceName(spans[0]); got != "fallback-name" {
		t.Errorf("service.name = %q, want Config fallback", got)
	}
}

func TestSetup_EnvServiceNameWins(t *testing.T) {
	clearOTelEnv(t)
	t.Setenv("OTEL_SERVICE_NAME", "from-env")
	exp := &keepingExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
	shutdown := setup(t, Config{ServiceName: "fallback-name", SpanExporter: exp})

	_, span := otelapi.Tracer("test").Start(context.Background(), "op")
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	if got := serviceName(spans[0]); got != "from-env" {
		t.Errorf("service.name = %q, want OTEL_SERVICE_NAME", got)
	}
}

func TestSetup_AttributesAreFallbacks(t *testing.T) {
	clearOTelEnv(t)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment.name=from-env")
	exp := &keepingExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
	shutdown := setup(t, Config{
		ServiceName:  "svc",
		SpanExporter: exp,
		Attributes: []attribute.KeyValue{
			attribute.String("service.version", "1.2.3"),
			attribute.String("deployment.environment.name", "fallback"),
		},
	})

	_, span := otelapi.Tracer("test").Start(context.Background(), "op")
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	got := map[attribute.Key]string{}
	for _, kv := range spans[0].Resource.Attributes() {
		got[kv.Key] = kv.Value.Emit()
	}
	if got["service.version"] != "1.2.3" {
		t.Errorf("service.version = %q, want the Config value", got["service.version"])
	}
	if got["deployment.environment.name"] != "from-env" {
		t.Errorf("deployment.environment.name = %q, want OTEL_RESOURCE_ATTRIBUTES", got["deployment.environment.name"])
	}
}

func TestSetup_MalformedResourceEnvIsNotFatal(t *testing.T) {
	clearOTelEnv(t)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "not-a-pair,service.version=1.2.3")
	exp := &keepingExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
	shutdown := setup(t, Config{ServiceName: "svc", SpanExporter: exp})

	_, span := otelapi.Tracer("test").Start(context.Background(), "op")
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	if spans := exp.GetSpans(); len(spans) != 1 || serviceName(spans[0]) != "svc" {
		t.Fatalf("want one span from service svc despite the malformed variable")
	}
}

func TestSetup_PropagatorIsTraceContextOnly(t *testing.T) {
	clearOTelEnv(t)
	shutdown := setup(t, Config{})
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	prop := otelapi.GetTextMapPropagator()
	if fields := prop.Fields(); len(fields) != 2 {
		t.Errorf("propagator fields = %v, want [traceparent tracestate]", fields)
	}

	ctx, span := otelapi.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	h := http.Header{}
	prop.Inject(ctx, propagation.HeaderCarrier(h))

	tp := h.Get("traceparent")
	if tp == "" || !strings.Contains(tp, TraceID(ctx)) {
		t.Errorf("traceparent = %q, want it to carry trace id %s", tp, TraceID(ctx))
	}
	if b := h.Get("baggage"); b != "" {
		t.Errorf("baggage header injected = %q, want none", b)
	}
}

func TestSetup_NoExporterStillCreatesValidIDs(t *testing.T) {
	clearOTelEnv(t)
	shutdown := setup(t, Config{})

	ctx, span := otelapi.Tracer("test").Start(context.Background(), "op")
	sc := span.SpanContext()
	span.End()

	if !sc.IsValid() || !sc.IsSampled() {
		t.Errorf("span context = %+v, want valid and sampled", sc)
	}
	if id := TraceID(ctx); len(id) != 32 || id == strings.Repeat("0", 32) {
		t.Errorf("TraceID() = %q, want 32 hex chars", id)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown() error = %v", err)
	}
}

func TestSetup_ViewsSetHistogramBuckets(t *testing.T) {
	clearOTelEnv(t)
	reg := prometheus.NewRegistry()
	shutdown := setup(t, Config{
		Registerer: reg,
		Views: []sdkmetric.View{sdkmetric.NewView(
			sdkmetric.Instrument{Name: "test.buckets"},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{Boundaries: []float64{1, 2}}},
		)},
	})
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	hist, err := otelapi.Meter("test").Float64Histogram("test.buckets")
	if err != nil {
		t.Fatalf("Float64Histogram() error = %v", err)
	}
	hist.Record(context.Background(), 1.5)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	for _, f := range families {
		if strings.HasPrefix(f.GetName(), "test_buckets") {
			if got := len(f.GetMetric()[0].GetHistogram().GetBucket()); got != 2 {
				t.Fatalf("buckets = %d, want 2 from the view", got)
			}
			return
		}
	}
	t.Fatalf("no test_buckets* family in %v", familyNames(families))
}

func TestSetup_MeterProviderWritesToRegisterer(t *testing.T) {
	clearOTelEnv(t)
	reg := prometheus.NewRegistry()
	shutdown := setup(t, Config{Registerer: reg})
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	hist, err := otelapi.Meter("test").Float64Histogram("test.latency", metric.WithUnit("s"))
	if err != nil {
		t.Fatalf("Float64Histogram() error = %v", err)
	}
	ctx, span := otelapi.Tracer("test").Start(context.Background(), "op")
	hist.Record(ctx, 0.25)
	span.End()

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	var mf *dto.MetricFamily
	for _, f := range families {
		if strings.HasPrefix(f.GetName(), "test_latency") {
			mf = f
		}
	}
	if mf == nil {
		t.Fatalf("no test_latency* family in %v", familyNames(families))
	}
	if mf.GetType() != dto.MetricType_HISTOGRAM {
		t.Fatalf("type = %v, want HISTOGRAM", mf.GetType())
	}
	h := mf.GetMetric()[0].GetHistogram()
	if h.GetSampleCount() != 1 || len(h.GetBucket()) == 0 {
		t.Fatalf("histogram = count %d, %d buckets; want 1 sample in explicit buckets",
			h.GetSampleCount(), len(h.GetBucket()))
	}

	var exemplarTrace string
	for _, b := range h.GetBucket() {
		for _, l := range b.GetExemplar().GetLabel() {
			if l.GetName() == "trace_id" {
				exemplarTrace = l.GetValue()
			}
		}
	}
	if exemplarTrace != TraceID(ctx) {
		t.Errorf("exemplar trace_id = %q, want %q", exemplarTrace, TraceID(ctx))
	}
}

func TestSetup_NoRegistererLeavesMeterProviderAlone(t *testing.T) {
	clearOTelEnv(t)
	before := otelapi.GetMeterProvider()
	shutdown := setup(t, Config{})
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	if after := otelapi.GetMeterProvider(); after != before {
		t.Error("Setup without a Registerer replaced the global meter provider")
	}
}

func TestBatchOptions_ZeroUsesSDKDefaults(t *testing.T) {
	if n := len(batchOptions(Config{})); n != 0 {
		t.Errorf("batchOptions(zero) = %d options, want 0", n)
	}
	if n := len(batchOptions(Config{MaxQueueSize: 1, MaxExportBatchSize: 1, BatchTimeout: 1, ExportTimeout: 1})); n != 4 {
		t.Errorf("batchOptions(all set) = %d options, want 4", n)
	}
}

func serviceName(s tracetest.SpanStub) string {
	for _, kv := range s.Resource.Attributes() {
		if kv.Key == "service.name" {
			return kv.Value.AsString()
		}
	}
	return ""
}

func familyNames(fs []*dto.MetricFamily) []string {
	names := make([]string, 0, len(fs))
	for _, f := range fs {
		names = append(names, f.GetName())
	}
	return names
}
