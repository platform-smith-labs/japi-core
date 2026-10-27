package router_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/platform-smith-labs/japi-core/v4/core"
	httpmiddleware "github.com/platform-smith-labs/japi-core/v4/middleware/http"
	"github.com/platform-smith-labs/japi-core/v4/otel"
	"github.com/platform-smith-labs/japi-core/v4/router"
)

const panicMark = "PANIC-VALUE-MARKER"

// assertSpanClean fails if s carries mark anywhere.
func assertSpanClean(t *testing.T, s sdktrace.ReadOnlySpan, mark string) {
	t.Helper()
	check := func(where, v string) {
		if strings.Contains(v, mark) {
			t.Errorf("span %q carries %s in %s", s.Name(), mark, where)
		}
	}
	check("name", s.Name())
	check("status", s.Status().Description)
	for _, kv := range s.Attributes() {
		check(string(kv.Key), kv.Value.Emit())
	}
	for _, e := range s.Events() {
		for _, kv := range e.Attributes {
			check(e.Name+"."+string(kv.Key), kv.Value.Emit())
		}
	}
}

func setDefaultLogger(t *testing.T, l *slog.Logger) {
	prev := slog.Default()
	slog.SetDefault(l)
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != core.ProblemContentType {
		t.Fatalf("Content-Type = %q, body %q", ct, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestPanicBecomesProblemWithTraceID(t *testing.T) {
	core.SetTypeBase("https://errors.example.test/")
	t.Cleanup(func() { core.SetTypeBase("") })

	for name, value := range map[string]any{
		"string": "boom " + panicMark,
		"error":  fmt.Errorf("wrapped %s", panicMark),
	} {
		t.Run(name, func(t *testing.T) {
			r, logger, buf, spans := privacyRouter(t)
			setDefaultLogger(t, logger)
			r.Get(routeWant, func(http.ResponseWriter, *http.Request) { panic(value) })

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/1", nil))

			if rec.Code != 500 {
				t.Fatalf("status %d", rec.Code)
			}
			body := decodeProblem(t, rec)
			ended := spans.Ended()
			if len(ended) != 1 {
				t.Fatalf("%d spans", len(ended))
			}
			span := ended[0]
			traceID := span.SpanContext().TraceID().String()
			if body["trace_id"] != traceID || rec.Header().Get("X-Request-ID") != traceID {
				t.Errorf("trace_id %v, X-Request-ID %q, span %s", body["trace_id"], rec.Header().Get("X-Request-ID"), traceID)
			}
			if body["type"] != "https://errors.example.test/internal_error" {
				t.Errorf("body %v", body)
			}
			if strings.Contains(rec.Body.String(), panicMark) {
				t.Errorf("panic value in body")
			}

			// Span: status pinned to "panic" (not overwritten by otelhttp's
			// status-from-code), stack recorded, no value anywhere.
			if span.Status().Code != codes.Error || span.Status().Description != otel.PanicStatusDescription {
				t.Errorf("span status %v %q", span.Status().Code, span.Status().Description)
			}
			assertSpanClean(t, span, panicMark)
			var stack bool
			for _, e := range span.Events() {
				for _, kv := range e.Attributes {
					if kv.Key == "exception.stacktrace" && strings.Contains(kv.Value.AsString(), "goroutine") {
						stack = true
					}
				}
			}
			if !stack {
				t.Errorf("no exception.stacktrace on the span: %+v", span.Events())
			}

			// Log: one ERROR line with the stack and trace ID, no value.
			out := buf.String()
			if strings.Contains(out, panicMark) {
				t.Errorf("panic value logged:\n%s", out)
			}
			line := findLine(t, out, "panic recovered")
			if line["level"] != "ERROR" || line["trace_id"] != traceID || line["exception.stacktrace"] == nil {
				t.Errorf("panic line %v", line)
			}
			if strings.Count(out, `"level":"ERROR"`) != 1 {
				t.Errorf("want exactly one ERROR line:\n%s", out)
			}
		})
	}
}

func TestPanicAfterResponseStartedWritesNothingMore(t *testing.T) {
	r, logger, _, _ := privacyRouter(t)
	setDefaultLogger(t, logger)
	r.Get(routeWant, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("partial"))
		panic(panicMark)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/1", nil))
	if rec.Code != http.StatusAccepted || rec.Body.String() != "partial" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestAbortHandlerIsReraised(t *testing.T) {
	r, logger, _, _ := privacyRouter(t)
	setDefaultLogger(t, logger)
	r.Get(routeWant, func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	defer func() {
		if p := recover(); p != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", p)
		}
	}()
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/1", nil))
}

// Without a Recoverer inside the span, Middleware itself keeps the value off
// the span (otelhttp ends the span with a bare defer, which would record it)
// and re-raises a constant.
func TestMiddlewareWithoutRecovererWithholdsPanicValue(t *testing.T) {
	spans := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
	h := otel.Middleware("test", otel.WithTracerProvider(tp))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(errors.New(panicMark))
	}))
	func() {
		defer func() {
			if p := recover(); p != otel.ErrHandlerPanic {
				t.Fatalf("re-raised %v, want otel.ErrHandlerPanic", p)
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	}()
	ended := spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("%d spans", len(ended))
	}
	assertSpanClean(t, ended[0], panicMark)
	if ended[0].Status().Description != otel.PanicStatusDescription {
		t.Errorf("status %q", ended[0].Status().Description)
	}
}

func TestServerProblemPinsCodeOnSpan(t *testing.T) {
	r, logger, _, spans := privacyRouter(t)
	setDefaultLogger(t, logger)
	r.Get(routeWant, router.AdaptErrorHandler(func(http.ResponseWriter, *http.Request) error {
		return core.Unavailable(core.CodeUnavailable, errors.New("dial tcp: "+panicMark))
	}))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/1", nil))
	if rec.Code != 503 || strings.Contains(rec.Body.String(), panicMark) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	s := spans.Ended()[0]
	if s.Status().Code != codes.Error || s.Status().Description != "unavailable" {
		t.Errorf("status %v %q, want Error unavailable", s.Status().Code, s.Status().Description)
	}
	for _, kv := range s.Attributes() {
		if kv.Key == "error.type" && kv.Value.AsString() != "unavailable" {
			t.Errorf("error.type = %q, want the code", kv.Value.AsString())
		}
	}
}

func TestRequestIDIsTraceID(t *testing.T) {
	r, _, _, spans := privacyRouter(t)
	var handlerID string
	r.Get(routeWant, func(_ http.ResponseWriter, req *http.Request) {
		handlerID = httpmiddleware.GetRequestID(req)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/1", nil)
	req.Header.Set("X-Request-ID", "caller-id-123\x01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	s := spans.Ended()[0]
	traceID := s.SpanContext().TraceID().String()
	if got := rec.Header().Get("X-Request-ID"); got != traceID {
		t.Errorf("X-Request-ID = %q, want trace ID %s", got, traceID)
	}
	if handlerID != traceID {
		t.Errorf("GetRequestID in handler = %q, want trace ID", handlerID)
	}
	var recorded []string
	for _, kv := range s.Attributes() {
		if kv.Key == "http.request.header.x-request-id" {
			recorded = kv.Value.AsStringSlice()
		}
	}
	if len(recorded) != 1 || recorded[0] != "caller-id-123" {
		t.Errorf("inbound X-Request-ID attribute = %v", recorded)
	}

	// With a traceparent, the inbound X-Request-ID is not recorded.
	prev := otelapi.GetTextMapPropagator()
	otelapi.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otelapi.SetTextMapPropagator(prev) })
	req = httptest.NewRequest(http.MethodGet, "/items/1", nil)
	req.Header.Set("X-Request-ID", "caller-id-456")
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("X-Request-ID = %q, want the continued trace ID", got)
	}
	for _, kv := range spans.Ended()[1].Attributes() {
		if kv.Key == "http.request.header.x-request-id" {
			t.Errorf("recorded inbound ID despite traceparent: %v", kv.Value.Emit())
		}
	}
}

func TestRequestIDWithoutTracingStaysChis(t *testing.T) {
	r := router.NewChiRouter()
	r.Get("/x", func(http.ResponseWriter, *http.Request) {})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "caller-id-789")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "caller-id-789" {
		t.Errorf("X-Request-ID = %q", got)
	}
}

func TestUnmatchedRequestsGetProblems(t *testing.T) {
	r := router.NewChiRouter()
	r.Get("/items/{id}", func(http.ResponseWriter, *http.Request) {})
	r.Route("/sub", func(sr chi.Router) {
		sr.Post("/x", func(http.ResponseWriter, *http.Request) {})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != 404 || decodeProblem(t, rec)["status"] != float64(404) {
		t.Fatalf("404: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/items/1", nil))
	if rec.Code != 405 || decodeProblem(t, rec)["status"] != float64(405) {
		t.Fatalf("405: %d %s", rec.Code, rec.Body.String())
	}
	if allow := rec.Header().Values("Allow"); len(allow) != 1 || allow[0] != http.MethodGet {
		t.Errorf("Allow = %v, want [GET]", allow)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sub/x", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("sub-router 405: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sub/nope", nil))
	if rec.Code != 404 {
		t.Fatalf("sub-router 404: %d", rec.Code)
	}
	decodeProblem(t, rec)
}
