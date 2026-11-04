// Package otlp builds an OTLP trace exporter (HTTP/protobuf) from the standard
// OTEL_EXPORTER_OTLP_* environment variables, for injection into
// otelsdk.Config.SpanExporter (package otel/sdk).
//
// It is a separate package so that services which do not export traces do not
// link the exporter and its transitive dependencies.
package otlp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const protocolHTTPProtobuf = "http/protobuf"

// FromEnv returns an OTLP HTTP/protobuf span exporter configured from the
// environment, or (nil, nil) when exporting is not configured.
//
// Exporting is configured when OTEL_EXPORTER_OTLP_TRACES_ENDPOINT or
// OTEL_EXPORTER_OTLP_ENDPOINT is non-empty, unless OTEL_TRACES_EXPORTER is
// "none". All other settings (headers, timeout, compression, TLS) are read by
// the exporter from the standard OTEL_EXPORTER_OTLP_* variables.
//
// Only the http/protobuf protocol is supported. If OTEL_EXPORTER_OTLP_PROTOCOL
// or OTEL_EXPORTER_OTLP_TRACES_PROTOCOL names a different protocol, FromEnv
// returns an error rather than silently exporting over the wrong one.
//
// Creating the exporter does not connect to the endpoint; an unreachable
// collector surfaces as export errors, never as a startup failure.
func FromEnv(ctx context.Context) (sdktrace.SpanExporter, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OTEL_TRACES_EXPORTER")), "none") {
		return nil, nil
	}
	if firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return nil, nil
	}
	if p := firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "OTEL_EXPORTER_OTLP_PROTOCOL"); p != "" && p != protocolHTTPProtobuf {
		return nil, fmt.Errorf("otlp: unsupported protocol %q (only %q is supported)", p, protocolHTTPProtobuf)
	}

	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("otlp: create trace exporter: %w", err)
	}
	return exp, nil
}

// firstNonEmpty returns the trimmed value of the first set, non-blank
// environment variable in keys, or "".
func firstNonEmpty(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
