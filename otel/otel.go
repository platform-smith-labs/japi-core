// Package otel instruments HTTP servers, HTTP clients, database/sql drivers
// and slog handlers with OpenTelemetry.
//
// It depends only on the OpenTelemetry API (plus otelhttp, chi and the
// standard library), never on the SDK: importing it links no exporter, starts
// no goroutine and registers nothing. Until the process installs an SDK
// (see the otel/sdk package, imported as otelsdk, and its Setup function) the
// OpenTelemetry globals are the API's no-op implementations, so instrumented
// code costs close to nothing.
//
// Contents:
//   - Middleware: server spans and the http.server.request.duration histogram
//   - Transport, NoPropagationTransport, InjectProxy: client spans and trace
//     context propagation
//   - WrapDriver, WrapConnector: database/sql tracing
//   - TraceHandler: slog trace correlation and attribute filtering
//   - TraceID: the trace ID of a context, for correlation
package otel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceID returns the hex-encoded trace ID of the span in ctx, or "" when ctx
// carries no span or the span's trace ID is invalid (for example when no SDK
// was installed and the global tracer is the no-op).
func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}
