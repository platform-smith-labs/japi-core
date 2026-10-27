package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/handler"
	httpmiddleware "github.com/platform-smith-labs/japi-core/v4/middleware/http"
	"github.com/platform-smith-labs/japi-core/v4/middleware/typed"
	"github.com/platform-smith-labs/japi-core/v4/otel"
	"github.com/platform-smith-labs/japi-core/v4/router"
)

// Values a request carries that must never reach a log line.
const (
	clientIP   = "203.0.113.77"
	userAgent  = "UA-MARKER/1.0"
	queryMark  = "QS-MARKER"
	pathMark   = "PATH-MARKER"
	requestID  = "RID-MARKER"
	routeWant  = "/items/{id}"
	requestURL = "/items/" + pathMark + "?secret=" + queryMark
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// privacyRouter builds the standard router with tracing and a JSON logger
// (through otel.TraceHandler) writing into the returned buffer.
func privacyRouter(t *testing.T) (chi.Router, *slog.Logger, *syncBuffer, *tracetest.SpanRecorder) {
	t.Helper()
	spans := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	buf := &syncBuffer{}
	logger := slog.New(otel.TraceHandler(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}), otel.LogConfig{}))

	r := router.NewChiRouterWithOptions(router.WithTracing("test", otel.WithTracerProvider(tp), otel.WithSkipPaths()))
	r.Use(httpmiddleware.WithLogging(logger))
	return r, logger, buf, spans
}

// serve sends one request carrying every marker and returns the log output.
func serve(t *testing.T, h http.Handler, buf *syncBuffer) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, requestURL, nil)
	req.RemoteAddr = clientIP + ":4711"
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("X-Real-IP", clientIP)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Request-ID", requestID)
	h.ServeHTTP(httptest.NewRecorder(), req)
	return buf.String()
}

func assertNoLeaks(t *testing.T, out string) {
	t.Helper()
	for _, leak := range []string{clientIP, userAgent, queryMark, pathMark, requestID,
		`"request_id"`, `"remote_addr"`, `"user_agent"`, `"query"`, `"path"`} {
		if strings.Contains(out, leak) {
			t.Errorf("log output contains %s:\n%s", leak, out)
		}
	}
}

// findLine returns the first decoded log line with the given msg.
func findLine(t *testing.T, out, msg string) map[string]any {
	t.Helper()
	return findLineWith(t, out, msg, "")
}

// findLineWith returns the first decoded log line with the given msg that
// also has key (any key when key is "").
func findLineWith(t *testing.T, out, msg, key string) map[string]any {
	t.Helper()
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("decode %q: %v", l, err)
		}
		if _, ok := m[key]; m["msg"] == msg && (key == "" || ok) {
			return m
		}
	}
	t.Fatalf("no %q line in:\n%s", msg, out)
	return nil
}

func assertCorrelated(t *testing.T, line map[string]any, spans *tracetest.SpanRecorder) {
	t.Helper()
	ended := spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("want 1 server span, got %d", len(ended))
	}
	sc := ended[0].SpanContext()
	if line["trace_id"] != sc.TraceID().String() {
		t.Errorf("trace_id = %v, want %s", line["trace_id"], sc.TraceID())
	}
	if line["span_id"] != sc.SpanID().String() {
		t.Errorf("span_id = %v, want %s", line["span_id"], sc.SpanID())
	}
	if line["trace_flags"] != "01" {
		t.Errorf("trace_flags = %v, want 01", line["trace_flags"])
	}
}

func TestAccessLog_CorrelatedAndPrivate(t *testing.T) {
	h, _, buf, spans := privacyRouter(t)
	h.Get(routeWant, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	out := serve(t, h, buf)
	assertNoLeaks(t, out)

	resp := findLine(t, out, "HTTP Response")
	assertCorrelated(t, resp, spans)
	if resp["http.route"] != routeWant {
		t.Errorf("http.route = %v, want %s", resp["http.route"], routeWant)
	}
	if resp["status"] != float64(http.StatusNoContent) {
		t.Errorf("status = %v", resp["status"])
	}
	assertCorrelated(t, findLine(t, out, "HTTP Request"), spans)
}

func TestAccessLog_UnmatchedRoute(t *testing.T) {
	h, _, buf, _ := privacyRouter(t)
	// chi skips the middleware stack entirely on a mux with no routes.
	h.Get("/other", func(http.ResponseWriter, *http.Request) {})
	out := serve(t, h, buf)
	assertNoLeaks(t, out)
	if got := findLine(t, out, "HTTP Response")["http.route"]; got != otel.UnmatchedRoute {
		t.Errorf("http.route = %v, want %s", got, otel.UnmatchedRoute)
	}
}

func TestTypedLogging_CorrelatedAndPrivate(t *testing.T) {
	h, logger, buf, spans := privacyRouter(t)

	var handlerRequestID string
	typedHandler := func(ctx handler.HandlerContext[struct{}, struct{}], _ http.ResponseWriter, _ *http.Request) (struct{}, error) {
		handlerRequestID = ctx.RequestID.ValueOrDefault()
		ctx.Logger.Info("in handler")
		return struct{}{}, errors.New("boom")
	}
	h.Get(routeWant, handler.AdaptHandler(nil, logger, typed.WithLogging(typedHandler)))

	out := serve(t, h, buf)
	assertNoLeaks(t, out)

	if handlerRequestID == "" {
		t.Error("ctx.RequestID no longer populated by the adapter")
	}
	for _, msg := range []string{"AdaptHandler creating context", "HTTP Request", "HTTP Response Error", "in handler", "Handler error"} {
		// The access log middleware also logs "HTTP Request"; the typed
		// middleware's line is the one with start_time.
		key := ""
		if strings.HasPrefix(msg, "HTTP ") {
			key = "start_time"
		}
		line := findLineWith(t, out, msg, key)
		assertCorrelated(t, line, spans)
		if msg != "in handler" && line["http.route"] != routeWant {
			t.Errorf("%s: http.route = %v, want %s", msg, line["http.route"], routeWant)
		}
	}
}

func TestWriteAPIError_LogCorrelatedAndPrivate(t *testing.T) {
	h, _, buf, spans := privacyRouter(t)

	prev := slog.Default()
	slog.SetDefault(slog.New(otel.TraceHandler(slog.NewJSONHandler(buf, nil), otel.LogConfig{})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	h.Get(routeWant, router.AdaptErrorHandler(func(w http.ResponseWriter, _ *http.Request) error {
		return core.NewAPIError(http.StatusNotFound, "not found")
	}))

	out := serve(t, h, buf)
	assertNoLeaks(t, out)
	line := findLine(t, out, "API error response")
	assertCorrelated(t, line, spans)
	if line["http.route"] != routeWant || line["http.request.method"] != http.MethodGet {
		t.Errorf("request context wrong: %v", line)
	}
}
