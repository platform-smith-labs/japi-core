package otel

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type harness struct {
	spans  *tracetest.SpanRecorder
	reader *sdkmetric.ManualReader
	router chi.Router
}

func newHarness(t *testing.T, opts ...MiddlewareOption) *harness {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})

	r := chi.NewRouter()
	r.Use(Middleware("test-service", append([]MiddlewareOption{WithTracerProvider(tp), WithMeterProvider(mp)}, opts...)...))
	ok := func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }
	r.Get("/users/{id}", ok)
	r.Get("/health", ok)
	r.Get("/metrics", ok)
	r.Get("/stream", ok)
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, brw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer conn.Close()
		_, _ = brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = brw.Flush()
		// Keep "serving" the connection after the handshake.
		time.Sleep(100 * time.Millisecond)
	})
	return &harness{spans: sr, reader: reader, router: r}
}

func (h *harness) serve(method, target string, hdr http.Header) {
	req := httptest.NewRequest(method, target, nil)
	req.RemoteAddr = "203.0.113.7:51234"
	req.Header.Set("User-Agent", "probe/1.0")
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	for k, v := range hdr {
		req.Header[k] = v
	}
	h.router.ServeHTTP(httptest.NewRecorder(), req)
}

// durationRoutes returns the http.route of each duration data point.
func (h *harness) durationRoutes(t *testing.T) map[string]uint64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := h.reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	out := map[string]uint64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.server.request.duration" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
				route, _ := dp.Attributes.Value("http.route")
				out[route.AsString()] += dp.Count
			}
		}
	}
	return out
}

func TestMiddleware_SpanNamedAfterRoutePattern(t *testing.T) {
	h := newHarness(t)
	h.serve(http.MethodGet, "/users/42", nil)

	spans := h.spans.Ended()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	// Convention: "{METHOD} {route}" per the OpenTelemetry HTTP span-name rule.
	if got := spans[0].Name(); got != "GET /users/{id}" {
		t.Errorf("span name = %q, want %q", got, "GET /users/{id}")
	}
	attrs := attribute.NewSet(spans[0].Attributes()...)
	if v, _ := attrs.Value("http.route"); v.AsString() != "/users/{id}" {
		t.Errorf("http.route = %q", v.AsString())
	}
	for _, kv := range spans[0].Attributes() {
		if _, ok := serverSpanAttributes[kv.Key]; !ok {
			t.Errorf("attribute outside the allowlist on span: %s=%s", kv.Key, kv.Value.Emit())
		}
		for _, leaked := range []string{"203.0.113.7", "198.51.100.9", "probe/1.0"} {
			if strings.Contains(kv.Value.Emit(), leaked) {
				t.Errorf("attribute %s leaks client data %q", kv.Key, leaked)
			}
		}
	}
	if spans[0].SpanKind().String() != "server" {
		t.Errorf("span kind = %s, want server", spans[0].SpanKind())
	}
}

func TestMiddleware_UnmatchedPath(t *testing.T) {
	h := newHarness(t)
	h.serve(http.MethodGet, "/no/such/path/123", nil)

	spans := h.spans.Ended()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	if got := spans[0].Name(); got != "GET unmatched" {
		t.Errorf("span name = %q, want %q", got, "GET unmatched")
	}
	for _, kv := range spans[0].Attributes() {
		if kv.Key == "http.route" && kv.Value.AsString() != UnmatchedRoute {
			t.Errorf("http.route = %q", kv.Value.AsString())
		}
	}
	if routes := h.durationRoutes(t); routes[UnmatchedRoute] != 1 || len(routes) != 1 {
		t.Errorf("duration routes = %v, want only %q", routes, UnmatchedRoute)
	}
}

func TestMiddleware_SkipsHealthAndMetrics(t *testing.T) {
	h := newHarness(t)
	h.serve(http.MethodGet, "/health", nil)
	h.serve(http.MethodGet, "/metrics", nil)

	if n := len(h.spans.Ended()) + len(h.spans.Started()); n != 0 {
		t.Errorf("want no spans for skipped paths, got %d", n)
	}
	if routes := h.durationRoutes(t); len(routes) != 0 {
		t.Errorf("want no duration points for skipped paths, got %v", routes)
	}
}

func TestMiddleware_SkipPathsSkipOnlyReads(t *testing.T) {
	h := newHarness(t)
	h.serve(http.MethodHead, "/health", nil)
	if n := len(h.spans.Ended()) + len(h.spans.Started()); n != 0 {
		t.Fatalf("HEAD /health: want no span, got %d", n)
	}
	// Not a probe: the 405 it gets must lead to a trace.
	h.serve(http.MethodPost, "/health", nil)
	h.serve(http.MethodDelete, "/metrics", nil)
	spans := h.spans.Ended()
	if len(spans) != 2 {
		t.Fatalf("want 2 ended spans for non-read requests to skip paths, got %d", len(spans))
	}
	for _, s := range spans {
		var status int64
		for _, kv := range s.Attributes() {
			if kv.Key == "http.response.status_code" {
				status = kv.Value.AsInt64()
			}
		}
		if status != http.StatusMethodNotAllowed {
			t.Errorf("span %q status = %d, want 405", s.Name(), status)
		}
	}
}

func TestMiddleware_WithSkipPathsReplacesDefaults(t *testing.T) {
	h := newHarness(t, WithSkipPaths("/users/1"))
	h.serve(http.MethodGet, "/health", nil)
	h.serve(http.MethodGet, "/users/1", nil)

	spans := h.spans.Ended()
	if len(spans) != 1 || spans[0].Name() != "GET /health" {
		t.Fatalf("want only the /health span, got %d spans", len(spans))
	}
}

func TestMiddleware_DurationExclusions(t *testing.T) {
	h := newHarness(t, ExcludeFromDuration("/stream"))
	h.serve(http.MethodGet, "/users/7", nil)
	h.serve(http.MethodGet, "/stream", nil)

	routes := h.durationRoutes(t)
	if routes["/users/{id}"] != 1 {
		t.Errorf("want 1 duration point for /users/{id}, got %v", routes)
	}
	if _, ok := routes["/stream"]; ok {
		t.Errorf("excluded route recorded: %v", routes)
	}
	// The excluded route is still traced.
	if len(h.spans.Ended()) != 2 {
		t.Errorf("want 2 spans, got %d", len(h.spans.Ended()))
	}
}

func TestMiddleware_WebSocketUpgrade(t *testing.T) {
	h := newHarness(t)
	srv := httptest.NewServer(h.router)
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("GET /ws HTTP/1.1\r\nHost: x\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n" +
		"Sec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"))
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// The handler is still holding the connection, but the span has ended.
	deadline := time.Now().Add(time.Second)
	for len(h.spans.Ended()) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	spans := h.spans.Ended()
	if len(spans) != 1 {
		t.Fatalf("want span ended at handshake, got %d ended spans", len(spans))
	}
	if spans[0].Name() != "GET /ws" {
		t.Errorf("span name = %q", spans[0].Name())
	}
	if spans[0].EndTime().Sub(spans[0].StartTime()) >= 100*time.Millisecond {
		t.Errorf("span lasted the connection (%v), want it to end at the handshake",
			spans[0].EndTime().Sub(spans[0].StartTime()))
	}

	srv.Close() // waits for the handler to return
	if routes := h.durationRoutes(t); len(routes) != 0 {
		t.Errorf("websocket upgrade recorded in duration histogram: %v", routes)
	}
}

func TestIsWebSocketUpgrade(t *testing.T) {
	cases := []struct {
		conn, upgrade string
		want          bool
	}{
		{"Upgrade", "websocket", true},
		{"keep-alive, Upgrade", "WebSocket", true},
		{"keep-alive", "websocket", false},
		{"Upgrade", "h2c", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Connection", c.conn)
		r.Header.Set("Upgrade", c.upgrade)
		if got := isWebSocketUpgrade(r); got != c.want {
			t.Errorf("Connection=%q Upgrade=%q: got %v", c.conn, c.upgrade, got)
		}
	}
}

// TestMiddleware_SpanCarriesNoRequestData sends identifiers in the path and
// query and asserts that neither reaches any part of the span, and that every
// attribute key is on the allowlist (so a dependency upgrade that adds a key
// fails here).
func TestMiddleware_SpanCarriesNoRequestData(t *testing.T) {
	const pathMarker = "PATHMARKER-user@example.com"
	const queryMarker = "QUERYMARKER-secret-token"
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	r := chi.NewRouter()
	r.Use(Middleware("test-service", WithTracerProvider(tp)))
	r.Post("/users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	for _, target := range []string{
		"/users/" + pathMarker + "?token=" + queryMarker,
		"/nope/" + pathMarker + "?token=" + queryMarker,
	} {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader("body"))
		req.Host = pathMarker + ".example.com"
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	spans := sr.Ended()
	if len(spans) != 2 {
		t.Fatalf("want 2 spans, got %d", len(spans))
	}
	leaks := func(where, v string) {
		for _, m := range []string{pathMarker, queryMarker} {
			if strings.Contains(v, m) {
				t.Errorf("%s contains %q: %q", where, m, v)
			}
		}
	}
	for _, s := range spans {
		leaks("span name", s.Name())
		leaks("status description", s.Status().Description)
		for _, kv := range s.Attributes() {
			if _, ok := serverSpanAttributes[kv.Key]; !ok {
				t.Errorf("attribute key %q is not on the allowlist", kv.Key)
			}
			leaks("attribute "+string(kv.Key), kv.Value.Emit())
		}
		for _, ev := range s.Events() {
			leaks("event name", ev.Name)
			for _, kv := range ev.Attributes {
				leaks("event attribute "+string(kv.Key), kv.Value.Emit())
			}
		}
	}
	if got := spans[0].Name(); got != "POST /users/{id}" {
		t.Errorf("span name = %q, want %q", got, "POST /users/{id}")
	}
	if got := spans[1].Name(); got != "POST "+UnmatchedRoute {
		t.Errorf("span name = %q, want %q", got, "POST "+UnmatchedRoute)
	}
}

var errFlushFailed = errors.New("flush failed")

type failingFlushWriter struct{ *httptest.ResponseRecorder }

func (w failingFlushWriter) FlushError() error { return errFlushFailed }

func TestMiddleware_FlushErrorReachesHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Middleware("test-service", WithTracerProvider(sdktrace.NewTracerProvider())))
	var got error
	r.Get("/stream", func(w http.ResponseWriter, _ *http.Request) {
		got = http.NewResponseController(w).Flush()
	})
	r.ServeHTTP(failingFlushWriter{httptest.NewRecorder()}, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if !errors.Is(got, errFlushFailed) {
		t.Errorf("ResponseController.Flush() = %v, want %v", got, errFlushFailed)
	}
}

func TestTracingWriter_FlushError(t *testing.T) {
	tw := &tracingWriter{ResponseWriter: failingFlushWriter{httptest.NewRecorder()}}
	if err := http.NewResponseController(tw).Flush(); !errors.Is(err, errFlushFailed) {
		t.Errorf("Flush() = %v, want %v", err, errFlushFailed)
	}
}

// TestMiddleware_ApplicationWritesAreKept verifies that the allowlist filters
// only what the instrumentation records: a later middleware and the handler
// get the raw span from the request context, and their attributes, errors
// and events reach the exported span. The name still follows the route.
func TestMiddleware_ApplicationWritesAreKept(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	r := chi.NewRouter()
	r.Use(Middleware("test-service", WithTracerProvider(tp)))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace.SpanFromContext(r.Context()).SetAttributes(attribute.String("app.mw", "m"))
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("app.key", "v"))
		span.RecordError(errors.New("boom"))
		span.AddEvent("app.event")
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/42", nil))

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	s := spans[0]
	attrs := attribute.NewSet(s.Attributes()...)
	for k, want := range map[attribute.Key]string{"app.key": "v", "app.mw": "m", "http.route": "/users/{id}"} {
		if v, ok := attrs.Value(k); !ok || v.AsString() != want {
			t.Errorf("%s = %q (present %v), want %q", k, v.AsString(), ok, want)
		}
	}
	var gotErr, gotEvent bool
	for _, ev := range s.Events() {
		switch ev.Name {
		case "exception":
			for _, kv := range ev.Attributes {
				if kv.Key == "exception.message" && kv.Value.AsString() == "boom" {
					gotErr = true
				}
			}
		case "app.event":
			gotEvent = true
		}
	}
	if !gotErr || !gotEvent {
		t.Errorf("recorded error kept = %v, event kept = %v; events = %v", gotErr, gotEvent, s.Events())
	}
	if s.Name() != "GET /users/{id}" {
		t.Errorf("span name = %q", s.Name())
	}
}
