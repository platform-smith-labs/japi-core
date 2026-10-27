package otel

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

// clientSpanAttributes is the allowlist of attributes a client span may
// carry. otelhttp also sets url.full (path and query), server.port,
// user_agent.original and network.protocol.*; everything not listed here is
// dropped, so the URL path, query string and headers never reach a span.
var clientSpanAttributes = map[attribute.Key]struct{}{
	"http.request.method":       {},
	"server.address":            {},
	"http.response.status_code": {},
	"error.type":                {},
}

// clientErrorDescription replaces the span status description otelhttp sets
// from err.Error(): a *url.Error message contains the full request URL.
const clientErrorDescription = "HTTP request failed"

// Transport returns an http.RoundTripper that traces outgoing requests and
// injects the current trace context (traceparent/tracestate) into them, so the
// callee's server span joins the caller's trace. Use it for calls between
// your own services. For third-party destinations use NoPropagationTransport.
// A nil base uses http.DefaultTransport.
//
// Each request gets a client span named "{METHOD}" carrying only
// http.request.method, server.address (the host, without port),
// http.response.status_code and, on failure, error.type. The URL path, query
// string and headers are never recorded. A failed request's status
// description is a fixed string, not the error text (which would contain the
// URL).
//
// Each request is also recorded in the http.client.request.duration
// histogram (seconds), labelled with http.request.method, server.address,
// http.response.status_code and, on failure, error.type. server.address is a
// label, so use the transport for a bounded set of destination hosts.
//
// The tracer, meter and propagator are the globals, resolved per request, so
// a later otelsdk.Setup takes effect.
func Transport(base http.RoundTripper) http.RoundTripper {
	return newClientTransport(base, otelapi.GetTextMapPropagator())
}

// NoPropagationTransport is Transport without trace context injection: the
// call is traced (a client span and a duration measurement exactly as with
// Transport) but no traceparent or tracestate header is sent. Use it for
// requests to third parties, which have no use for your trace IDs.
// A nil base uses http.DefaultTransport.
func NoPropagationTransport(base http.RoundTripper) http.RoundTripper {
	return newClientTransport(base, propagation.NewCompositeTextMapPropagator())
}

// InjectProxy writes the trace context of r.Context() into r.Header using the
// global propagator, replacing any trace headers r already carries. It is
// meant for the outgoing request of an httputil.ReverseProxy whose handler
// runs inside Middleware, so the upstream joins the proxy's server span:
//
//	proxy := &httputil.ReverseProxy{
//	    Rewrite: func(pr *httputil.ProxyRequest) {
//	        pr.SetURL(upstream)
//	        otel.InjectProxy(pr.Out)
//	    },
//	}
//
// The inbound trace headers are removed first, so a header supplied by the
// caller is never forwarded verbatim when r.Context() carries no span.
func InjectProxy(r *http.Request) {
	// The W3C headers are removed explicitly: before an SDK is installed the
	// global propagator is a no-op that reports no Fields.
	for _, f := range proxyTraceHeaders {
		r.Header.Del(f)
	}
	prop := otelapi.GetTextMapPropagator()
	for _, f := range prop.Fields() {
		r.Header.Del(f)
	}
	prop.Inject(r.Context(), propagation.HeaderCarrier(r.Header))
}

// proxyTraceHeaders are always removed by InjectProxy, whatever the global
// propagator reports.
var proxyTraceHeaders = []string{"traceparent", "tracestate", "baggage"}

// clientTransport records the duration histogram around an otelhttp
// transport. otelhttp's own instruments are disabled: they would also label
// by server.port, url.scheme and network.protocol.*.
type clientTransport struct {
	traced http.RoundTripper
}

func newClientTransport(base http.RoundTripper, prop propagation.TextMapPropagator) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &clientTransport{
		traced: otelhttp.NewTransport(base,
			otelhttp.WithTracerProvider(clientTracerProvider{}),
			otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
			otelhttp.WithPropagators(prop),
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return spanMethod(r.Method)
			}),
		),
	}
}

func (t *clientTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.traced.RoundTrip(r)

	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", metricMethod(r.Method)),
		attribute.String("server.address", requestHost(r)),
	}
	if err != nil {
		attrs = append(attrs, attribute.String("error.type", fmt.Sprintf("%T", err)))
	} else {
		attrs = append(attrs, attribute.Int("http.response.status_code", resp.StatusCode))
	}
	clientDuration().Record(r.Context(), time.Since(start).Seconds(),
		metric.WithAttributeSet(attribute.NewSet(attrs...)))
	return resp, err
}

// clientDuration returns the http.client.request.duration histogram from the
// current global meter provider. The SDK caches instruments, so repeated
// calls return the same one.
func clientDuration() metric.Float64Histogram {
	h, err := otelapi.GetMeterProvider().Meter("github.com/platform-smith-labs/japi-core/v3/otel").Float64Histogram(
		"http.client.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of HTTP client requests."),
		metric.WithExplicitBucketBoundaries(requestDurationBuckets...),
	)
	if err != nil {
		otelapi.Handle(err)
		h, _ = metricnoop.NewMeterProvider().Meter("").Float64Histogram("")
	}
	return h
}

// requestHost is the destination host without port, lower-cased.
func requestHost(r *http.Request) string {
	h := ""
	if r.URL != nil {
		h = r.URL.Hostname()
	}
	if h == "" {
		h = r.Host
		if i := strings.LastIndexByte(h, ':'); i >= 0 && !strings.Contains(h[i:], "]") {
			h = h[:i]
		}
		h = strings.Trim(h, "[]")
	}
	return strings.ToLower(h)
}

// clientTracerProvider hands out tracers whose spans keep only the
// clientSpanAttributes allowlist. otelhttp sets the request attributes after
// the span has started, so filtering happens on the span, not at Start.
type clientTracerProvider struct {
	embedded.TracerProvider
}

func (p clientTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return clientTracer{name: name, opts: opts}
}

type clientTracer struct {
	embedded.Tracer
	name string
	opts []trace.TracerOption
}

func (t clientTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)
	clean := []trace.SpanStartOption{
		trace.WithAttributes(allowedClientAttributes(cfg.Attributes())...),
		trace.WithSpanKind(cfg.SpanKind()),
	}
	if ts := cfg.Timestamp(); !ts.IsZero() {
		clean = append(clean, trace.WithTimestamp(ts))
	}
	if links := cfg.Links(); len(links) > 0 {
		clean = append(clean, trace.WithLinks(links...))
	}
	if cfg.NewRoot() {
		clean = append(clean, trace.WithNewRoot())
	}
	ctx, span := otelapi.GetTracerProvider().Tracer(t.name, t.opts...).Start(ctx, spanName, clean...)
	// otelhttp writes through the returned clientSpan (filtered); the
	// context it hands to the base transport carries the raw span, so an
	// inner transport the application chose can record on it.
	return trace.ContextWithSpan(ctx, span), clientSpan{Span: span}
}

func allowedClientAttributes(in []attribute.KeyValue) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(in))
	for _, kv := range in {
		if _, ok := clientSpanAttributes[kv.Key]; ok {
			out = append(out, kv)
		}
	}
	return out
}

// clientSpan filters what otelhttp writes to a client span.
type clientSpan struct {
	trace.Span
}

func (s clientSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.Span.SetAttributes(allowedClientAttributes(kv)...)
}

func (s clientSpan) SetStatus(code codes.Code, description string) {
	if code == codes.Error && description != "" {
		description = clientErrorDescription
	}
	s.Span.SetStatus(code, description)
}

// RecordError is dropped: the error text of a failed request contains the
// URL. The failure is still visible through the status and error.type.
func (s clientSpan) RecordError(error, ...trace.EventOption) {}

// AddEvent is dropped: otelhttp events may carry request details.
func (s clientSpan) AddEvent(string, ...trace.EventOption) {}
