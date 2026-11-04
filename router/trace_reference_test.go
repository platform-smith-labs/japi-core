package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/otel"
	"github.com/platform-smith-labs/japi-core/v4/router"
)

// TestEveryProblemTraceIDIsExported pins the support-reference contract: the
// trace_id of any Problem Details body, including answers given without a
// handler (unmatched route, wrong method, a middleware refusing the request
// before routing, a non-read request to a skipped probe path), is the trace
// ID of a server span that has ENDED and been EXPORTED, and equals the
// X-Request-ID response header.
func TestEveryProblemTraceIDIsExported(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	r := router.NewChiRouterWithOptions(router.WithTracing("test", otel.WithTracerProvider(tp)))
	// A refusal before routing, mounted the way services mount auth.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("X-Refuse") != "" {
				_ = core.WriteProblem(w, req, core.Unauthorized(core.CodeUnauthorized, "Missing credentials."))
				return
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/health", func(http.ResponseWriter, *http.Request) {})
	r.Get("/items/{id}", func(http.ResponseWriter, *http.Request) {})
	r.Route("/admin", func(ar chi.Router) {
		ar.Use(func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				_ = core.WriteProblem(w, req, core.Unauthorized(core.CodeUnauthorized, "Missing credentials."))
			})
		})
		ar.Get("/things", func(http.ResponseWriter, *http.Request) {})
	})

	cases := []struct {
		name, method, target string
		refuse               bool
		status               int
	}{
		{"unmatched route", http.MethodGet, "/nope", false, http.StatusNotFound},
		{"wrong method", http.MethodDelete, "/items/1", false, http.StatusMethodNotAllowed},
		{"refused before routing", http.MethodGet, "/items/1", true, http.StatusUnauthorized},
		{"refused by sub-router middleware", http.MethodGet, "/admin/things", false, http.StatusUnauthorized},
		{"non-read request to a skip path", http.MethodPost, "/health", false, http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exp.Reset()
			req := httptest.NewRequest(tc.method, tc.target, nil)
			if tc.refuse {
				req.Header.Set("X-Refuse", "1")
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.status, rec.Body.String())
			}
			body := decodeProblem(t, rec)
			bodyID, _ := body["trace_id"].(string)
			if bodyID == "" {
				t.Fatalf("problem has no trace_id: %v", body)
			}
			if got := rec.Header().Get("X-Request-ID"); got != bodyID {
				t.Errorf("X-Request-ID = %q, body trace_id = %q", got, bodyID)
			}
			var server []tracetest.SpanStub
			for _, s := range exp.GetSpans() {
				if s.SpanKind == trace.SpanKindServer {
					server = append(server, s)
				}
			}
			if len(server) != 1 {
				t.Fatalf("want 1 exported server span, got %d", len(server))
			}
			s := server[0]
			if s.EndTime.IsZero() {
				t.Errorf("exported span %q has not ended", s.Name)
			}
			if got := s.SpanContext.TraceID().String(); got != bodyID {
				t.Errorf("exported span trace ID %s != body trace_id %s", got, bodyID)
			}
			if !s.SpanContext.IsSampled() {
				t.Errorf("exported span %q is not sampled", s.Name)
			}
		})
	}
}
