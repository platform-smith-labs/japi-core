package metrics

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	httpmiddleware "github.com/platform-smith-labs/japi-core/v4/middleware/http"
	"github.com/platform-smith-labs/japi-core/v4/otel"
	"github.com/platform-smith-labs/japi-core/v4/router"
)

var errFlush = errors.New("flush failed")

// failingFlushWriter is a ResponseWriter whose flush fails.
type failingFlushWriter struct{ *httptest.ResponseRecorder }

func (w failingFlushWriter) FlushError() error { return errFlush }

// TestFlushError_PropagatesThroughChain verifies that the tracing, logging and
// metrics wrappers report the underlying writer's flush error to
// http.ResponseController instead of swallowing it.
func TestFlushError_PropagatesThroughChain(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	r := router.NewChiRouterWithOptions(router.WithTracing("test", otel.WithTracerProvider(tp)))
	r.Use(httpmiddleware.WithLogging(slog.New(slog.NewTextHandler(io.Discard, nil))))
	enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), prometheus.NewRegistry())

	var got error
	r.Get("/stream", func(w http.ResponseWriter, _ *http.Request) {
		got = http.NewResponseController(w).Flush()
	})
	r.ServeHTTP(failingFlushWriter{httptest.NewRecorder()}, httptest.NewRequest(http.MethodGet, "/stream", nil))

	if !errors.Is(got, errFlush) {
		t.Errorf("ResponseController.Flush() = %v, want %v", got, errFlush)
	}
}

func TestResponseWriter_FlushError(t *testing.T) {
	rw := &responseWriter{ResponseWriter: failingFlushWriter{httptest.NewRecorder()}, statusCode: http.StatusOK}
	if err := http.NewResponseController(rw).Flush(); !errors.Is(err, errFlush) {
		t.Errorf("Flush() = %v, want %v", err, errFlush)
	}
}
