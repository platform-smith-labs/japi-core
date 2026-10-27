package otlp

import (
	"context"
	"testing"
)

// clearEnv blanks every variable FromEnv inspects so the host environment
// cannot leak into a test. t.Setenv restores the originals afterwards.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"OTEL_TRACES_EXPORTER",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
	} {
		t.Setenv(k, "")
	}
}

func TestFromEnv_NoEndpointReturnsNil(t *testing.T) {
	clearEnv(t)

	exp, err := FromEnv(context.Background())
	if err != nil {
		t.Fatalf("FromEnv() error = %v, want nil", err)
	}
	if exp != nil {
		t.Fatalf("FromEnv() = %v, want nil exporter", exp)
	}
}

func TestFromEnv_EndpointCreatesExporter(t *testing.T) {
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"} {
		t.Run(key, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(key, "http://127.0.0.1:4318")

			exp, err := FromEnv(context.Background())
			if err != nil {
				t.Fatalf("FromEnv() error = %v", err)
			}
			if exp == nil {
				t.Fatal("FromEnv() returned nil exporter with an endpoint set")
			}
			if err := exp.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})
	}
}

func TestFromEnv_TracesExporterNone(t *testing.T) {
	clearEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	t.Setenv("OTEL_TRACES_EXPORTER", "none")

	exp, err := FromEnv(context.Background())
	if err != nil || exp != nil {
		t.Fatalf("FromEnv() = (%v, %v), want (nil, nil)", exp, err)
	}
}

func TestFromEnv_RejectsOtherProtocols(t *testing.T) {
	clearEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	exp, err := FromEnv(context.Background())
	if err == nil {
		t.Fatal("FromEnv() error = nil, want unsupported protocol error")
	}
	if exp != nil {
		t.Fatalf("FromEnv() exporter = %v, want nil on error", exp)
	}
}

func TestFromEnv_AcceptsHTTPProtobuf(t *testing.T) {
	clearEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/protobuf")

	exp, err := FromEnv(context.Background())
	if err != nil || exp == nil {
		t.Fatalf("FromEnv() = (%v, %v), want exporter", exp, err)
	}
	_ = exp.Shutdown(context.Background())
}
