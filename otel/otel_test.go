package otel

import (
	"context"
	"os"
	"testing"

	otelapi "go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Recorded before any test installs a provider: what the globals look like
// when the package is merely imported.
var (
	importOnlyTracerProvider trace.TracerProvider
	importOnlyFields         []string
)

func TestMain(m *testing.M) {
	importOnlyTracerProvider = otelapi.GetTracerProvider()
	importOnlyFields = otelapi.GetTextMapPropagator().Fields()
	os.Exit(m.Run())
}

func TestNoSetup_GlobalsStayNoop(t *testing.T) {
	if _, ok := importOnlyTracerProvider.(*sdktrace.TracerProvider); ok {
		t.Fatal("global tracer provider is an SDK provider when the package is merely imported")
	}
	_, span := importOnlyTracerProvider.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	if span.SpanContext().IsValid() {
		t.Error("span from the import-only global has a valid span context; want no-op")
	}
	if len(importOnlyFields) != 0 {
		t.Errorf("global propagator fields = %v on import, want none", importOnlyFields)
	}
	if got := TraceID(context.Background()); got != "" {
		t.Errorf("TraceID(empty ctx) = %q, want \"\"", got)
	}
}

func TestTraceID_InvalidContext(t *testing.T) {
	ctx := trace.ContextWithSpanContext(context.Background(), trace.SpanContext{})
	if got := TraceID(ctx); got != "" {
		t.Errorf("TraceID(invalid) = %q, want \"\"", got)
	}
}
