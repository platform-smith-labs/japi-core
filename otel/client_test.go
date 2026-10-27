package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// installGlobals sets recording global providers and the Trace Context
// propagator for one test, restoring the previous globals afterwards.
func installGlobals(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder, *sdkmetric.ManualReader) {
	t.Helper()
	prevTP, prevMP, prevProp := otelapi.GetTracerProvider(), otelapi.GetMeterProvider(), otelapi.GetTextMapPropagator()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otelapi.SetTracerProvider(tp)
	otelapi.SetMeterProvider(mp)
	otelapi.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
		otelapi.SetTracerProvider(prevTP)
		otelapi.SetMeterProvider(prevMP)
		otelapi.SetTextMapPropagator(prevProp)
	})
	return tp, sr, reader
}

// headerServer serves a traced chi router that records the traceparent each
// request arrives with.
func headerServer(t *testing.T, tp trace.TracerProvider) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	r := chi.NewRouter()
	r.Use(Middleware("svc", WithTracerProvider(tp)))
	r.Get("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("traceparent"))
		mu.Unlock()
		_, _ = w.Write([]byte("ok"))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func spansOfKind(sr *tracetest.SpanRecorder, kind trace.SpanKind) []sdktrace.ReadOnlySpan {
	var out []sdktrace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if s.SpanKind() == kind {
			out = append(out, s)
		}
	}
	return out
}

func get(t *testing.T, rt http.RoundTripper, target string) {
	t.Helper()
	resp, err := (&http.Client{Transport: rt}).Get(target)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestTransport_PropagatesTraceToServer(t *testing.T) {
	tp, sr, _ := installGlobals(t)
	srv, seen := headerServer(t, tp)

	get(t, Transport(nil), srv.URL+"/items/42?token=SECRET-Q")

	if h := seen(); len(h) != 1 || h[0] == "" {
		t.Fatalf("server saw traceparent %q, want one non-empty", h)
	}
	clients, servers := spansOfKind(sr, trace.SpanKindClient), spansOfKind(sr, trace.SpanKindServer)
	if len(clients) != 1 || len(servers) != 1 {
		t.Fatalf("got %d client and %d server spans, want 1 and 1", len(clients), len(servers))
	}
	if c, s := clients[0], servers[0]; c.SpanContext().TraceID() != s.SpanContext().TraceID() {
		t.Errorf("trace IDs differ: client %s, server %s", c.SpanContext().TraceID(), s.SpanContext().TraceID())
	} else if s.Parent().SpanID() != c.SpanContext().SpanID() {
		t.Errorf("server span parent = %s, want client span %s", s.Parent().SpanID(), c.SpanContext().SpanID())
	}
}

func TestNoPropagationTransport_SendsNoTraceHeaders(t *testing.T) {
	tp, sr, _ := installGlobals(t)
	srv, seen := headerServer(t, tp)

	get(t, NoPropagationTransport(nil), srv.URL+"/items/42")

	if h := seen(); len(h) != 1 || h[0] != "" {
		t.Fatalf("server saw traceparent %q, want none", h)
	}
	clients, servers := spansOfKind(sr, trace.SpanKindClient), spansOfKind(sr, trace.SpanKindServer)
	if len(clients) != 1 {
		t.Fatalf("got %d client spans, want 1", len(clients))
	}
	if len(servers) == 1 && servers[0].SpanContext().TraceID() == clients[0].SpanContext().TraceID() {
		t.Error("server span joined the client trace without propagation")
	}
}

func TestTransport_ClientSpanAndMetricCarryNoURL(t *testing.T) {
	_, sr, reader := installGlobals(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/secret-path/7?token=SECRET-Q", nil)
	req.Header.Set("Authorization", "Bearer SECRET-H")
	req.Header.Set("User-Agent", "SECRET-UA")
	resp, err := (&http.Client{Transport: Transport(nil)}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	clients := spansOfKind(sr, trace.SpanKindClient)
	if len(clients) != 1 {
		t.Fatalf("got %d client spans, want 1", len(clients))
	}
	span := clients[0]
	if span.Name() != "GET" {
		t.Errorf("span name = %q, want GET", span.Name())
	}
	got := map[attribute.Key]string{}
	for _, kv := range span.Attributes() {
		got[kv.Key] = kv.Value.Emit()
	}
	for _, k := range []attribute.Key{"url.full", "url.path", "url.query", "user_agent.original", "server.port"} {
		if _, ok := got[k]; ok {
			t.Errorf("client span has %s", k)
		}
	}
	for k, v := range got {
		if strings.Contains(string(k), "header") || strings.Contains(v, "SECRET") {
			t.Errorf("client span attribute %s=%q leaks request details", k, v)
		}
	}
	if got["server.address"] != "127.0.0.1" || got["http.request.method"] != "GET" || got["http.response.status_code"] != "418" {
		t.Errorf("attributes = %v, want server.address, http.request.method and status", got)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	points := 0
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.client.request.duration" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
				points++
				for _, kv := range dp.Attributes.ToSlice() {
					switch kv.Key {
					case "http.request.method", "server.address", "http.response.status_code":
					default:
						t.Errorf("duration label %s not allowed", kv.Key)
					}
				}
			}
		}
	}
	if points != 1 {
		t.Errorf("got %d http.client.request.duration points, want 1", points)
	}
}

func TestTransport_FailedRequestHidesURLInStatus(t *testing.T) {
	_, sr, _ := installGlobals(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	target := srv.URL + "/secret-path?token=SECRET-Q"
	srv.Close() // connection refused

	if _, err := (&http.Client{Transport: Transport(nil)}).Get(target); err == nil {
		t.Fatal("expected an error from a closed server")
	}
	clients := spansOfKind(sr, trace.SpanKindClient)
	if len(clients) != 1 {
		t.Fatalf("got %d client spans, want 1", len(clients))
	}
	span := clients[0]
	if span.Status().Code != codes.Error {
		t.Errorf("status = %v, want Error", span.Status().Code)
	}
	if strings.Contains(span.Status().Description, "SECRET") || strings.Contains(span.Status().Description, "secret-path") {
		t.Errorf("status description leaks the URL: %q", span.Status().Description)
	}
	if len(span.Events()) != 0 {
		t.Errorf("span has %d events, want none", len(span.Events()))
	}
}

func TestInjectProxy_ForwardsCallerTrace(t *testing.T) {
	tp, sr, _ := installGlobals(t)
	var upstreamTP string
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		upstreamTP = r.Header.Get("traceparent")
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)

	proxy := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) {
		pr.SetURL(target)
		InjectProxy(pr.Out)
	}}
	r := chi.NewRouter()
	r.Use(Middleware("proxy", WithTracerProvider(tp)))
	r.Handle("/*", proxy)
	front := httptest.NewServer(r)
	defer front.Close()

	const callerTrace = "4bf92f3577b34da6a3ce929d0e0e4736"
	req, _ := http.NewRequest(http.MethodGet, front.URL+"/x", nil)
	req.Header.Set("traceparent", "00-"+callerTrace+"-00f067aa0ba902b7-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	parts := strings.Split(upstreamTP, "-")
	if len(parts) != 4 || parts[1] != callerTrace {
		t.Fatalf("upstream traceparent = %q, want trace %s", upstreamTP, callerTrace)
	}
	servers := spansOfKind(sr, trace.SpanKindServer)
	if len(servers) != 1 || parts[2] != servers[0].SpanContext().SpanID().String() {
		t.Errorf("upstream parent = %s, want the proxy's server span", parts[2])
	}
}

func TestInjectProxy_DropsInboundHeaderWithoutSpan(t *testing.T) {
	installGlobals(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set("tracestate", "k=v")
	InjectProxy(req)
	if req.Header.Get("traceparent") != "" || req.Header.Get("tracestate") != "" {
		t.Errorf("inbound trace headers forwarded: %v", req.Header)
	}
}

// TestInjectProxy_DropsInboundHeadersWithNoopPropagator covers a process that
// never installed an SDK: the global propagator reports no fields, yet the
// inbound trace headers must still not be forwarded.
func TestInjectProxy_DropsInboundHeadersWithNoopPropagator(t *testing.T) {
	prev := otelapi.GetTextMapPropagator()
	otelapi.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
	t.Cleanup(func() { otelapi.SetTextMapPropagator(prev) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set("tracestate", "k=v")
	req.Header.Set("baggage", "k=v")
	InjectProxy(req)
	for _, h := range []string{"traceparent", "tracestate", "baggage"} {
		if v := req.Header.Get(h); v != "" {
			t.Errorf("inbound %s forwarded: %q", h, v)
		}
	}
}

// attrTransport records an attribute on the span in the request context, as
// an application-chosen inner transport might.
type attrTransport struct{ base http.RoundTripper }

func (a attrTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	trace.SpanFromContext(r.Context()).SetAttributes(attribute.String("app.inner", "v"))
	return a.base.RoundTrip(r)
}

func TestTransport_InnerTransportWritesAreKept(t *testing.T) {
	tp, sr, _ := installGlobals(t)
	srv, _ := headerServer(t, tp)
	get(t, Transport(attrTransport{base: http.DefaultTransport}), srv.URL+"/x")

	clients := spansOfKind(sr, trace.SpanKindClient)
	if len(clients) != 1 {
		t.Fatalf("want 1 client span, got %d", len(clients))
	}
	set := attribute.NewSet(clients[0].Attributes()...)
	if v, ok := set.Value("app.inner"); !ok || v.AsString() != "v" {
		t.Errorf("app.inner = %q (present %v), want v", v.AsString(), ok)
	}
}
