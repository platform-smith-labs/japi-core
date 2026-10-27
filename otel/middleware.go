package otel

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

// UnmatchedRoute is the route used in span names, the http.route attribute and
// metric labels when no chi route pattern matched the request. The raw URL
// path is never used in its place: it is unbounded (cardinality) and may carry
// identifiers.
const UnmatchedRoute = "unmatched"

// DefaultSkipPaths are the request paths Middleware does not trace or measure
// unless WithSkipPaths replaces them: health/readiness probes and the metrics
// scrape endpoint, which would otherwise dominate trace volume.
var DefaultSkipPaths = []string{"/health", "/healthz", "/ready", "/readyz", "/live", "/livez", "/metrics"}

// requestDurationBuckets are the explicit bucket boundaries (seconds) the
// HTTP semantic conventions advise for http.server.request.duration.
var requestDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}

// serverSpanAttributes is the allowlist of attributes a server span may
// carry. otelhttp also records url.path (the raw path, which can carry
// identifiers), url.query, server.address/port (from the client-controlled
// Host header), client.address, network.peer.*, user_agent.original, body
// sizes and more, some of them only after the handler returns. Everything not
// listed here is dropped, at span start and on every later write, so a newer
// otelhttp that records another key cannot leak it.
var serverSpanAttributes = map[attribute.Key]struct{}{
	"http.request.method":       {},
	"url.scheme":                {},
	"network.protocol.version":  {},
	"http.route":                {},
	"http.response.status_code": {},
	"error.type":                {},
}

// serverErrorDescription replaces any span status description, which could
// otherwise quote request data.
const serverErrorDescription = "HTTP server error"

// MiddlewareOption configures Middleware.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	skipPaths       map[string]struct{}
	excludeDuration map[string]struct{}
	tracerProvider  trace.TracerProvider
	meterProvider   metric.MeterProvider
}

// WithSkipPaths replaces DefaultSkipPaths. Requests whose URL path exactly
// equals one of paths get no span and no duration measurement. Call with no
// arguments to trace every request.
func WithSkipPaths(paths ...string) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.skipPaths = make(map[string]struct{}, len(paths))
		for _, p := range paths {
			c.skipPaths[p] = struct{}{}
		}
	}
}

// ExcludeFromDuration names chi route patterns (e.g. "/events/stream") whose
// requests are not recorded in the http.server.request.duration histogram.
// Use it for long-lived responses such as server-sent events: their duration
// is the lifetime of a stream, not a latency, and would swamp the histogram.
// Their span is still created and lasts as long as the stream.
//
// WebSocket upgrade requests are always excluded and need not be listed.
func ExcludeFromDuration(routePatterns ...string) MiddlewareOption {
	return func(c *middlewareConfig) {
		for _, p := range routePatterns {
			c.excludeDuration[p] = struct{}{}
		}
	}
}

// WithTracerProvider sets the tracer provider. Default: the global provider,
// resolved on every request so a later otelsdk.Setup takes effect.
func WithTracerProvider(tp trace.TracerProvider) MiddlewareOption {
	return func(c *middlewareConfig) { c.tracerProvider = tp }
}

// WithMeterProvider sets the meter provider used for the request duration
// histogram. Default: the global provider.
func WithMeterProvider(mp metric.MeterProvider) MiddlewareOption {
	return func(c *middlewareConfig) { c.meterProvider = mp }
}

// Middleware returns server-side HTTP tracing middleware for chi routers,
// built on otelhttp. service is the otelhttp operation name.
//
// For every request that is not skipped (see WithSkipPaths) it:
//   - extracts the incoming W3C trace context and starts a server span;
//   - once the router has matched, names the span "{METHOD} {route}" (the
//     OpenTelemetry HTTP convention, e.g. "GET /users/{id}") and sets
//     http.route to the chi route pattern, or UnmatchedRoute when none
//     matched — never the raw path;
//   - keeps only an allowlist of attributes on the span
//     (http.request.method, url.scheme, network.protocol.version, http.route,
//     http.response.status_code, error.type): the raw URL path and query,
//     the Host header, client identity and user agent are never recorded,
//     and otelhttp's span events and recorded errors are dropped. The
//     allowlist applies only to what the instrumentation records:
//     trace.SpanFromContext(r.Context()) in a handler or later middleware
//     returns the unfiltered span, so attributes, errors and events the
//     application records itself are kept;
//   - records http.server.request.duration with http.route as a label,
//     except for ExcludeFromDuration routes and WebSocket upgrades.
//
// When the handler hijacks the connection (a WebSocket upgrade) the span ends
// at the hijack, i.e. at the 101 handshake, even if the handler keeps
// serving the connection afterwards.
//
// The middleware reads the route pattern after the inner handler returns, so
// it works at any position in a chi middleware stack (chi resolves the route
// only when the request reaches the router's mux).
func Middleware(service string, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := middlewareConfig{excludeDuration: map[string]struct{}{}}
	WithSkipPaths(DefaultSkipPaths...)(&cfg)
	for _, o := range opts {
		o(&cfg)
	}

	mp := cfg.meterProvider
	if mp == nil {
		mp = otelapi.GetMeterProvider()
	}
	duration, err := mp.Meter("github.com/platform-smith-labs/japi-core/v3/otel").Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of HTTP server requests."),
		metric.WithExplicitBucketBoundaries(requestDurationBuckets...),
	)
	if err != nil {
		otelapi.Handle(err)
		duration, _ = metricnoop.NewMeterProvider().Meter("").Float64Histogram("")
	}

	return func(next http.Handler) http.Handler {
		inner := &routeHandler{next: next, cfg: &cfg, duration: duration}
		traced := otelhttp.NewHandler(inner, service,
			otelhttp.WithTracerProvider(scrubbingTracerProvider{tp: cfg.tracerProvider}),
			// Duration is recorded by routeHandler so it can be skipped per
			// request and labelled with the resolved route. otelhttp's own
			// instruments would also label by server.address, which comes
			// from the client-controlled Host header.
			otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
			// Called at span start (route not yet known: "{METHOD}") and,
			// by otelhttp, again after the handler when the router has set
			// r.Pattern. Both paths agree with tracingWriter.finish.
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				if r.Pattern == "" {
					return spanMethod(r.Method)
				}
				return spanMethod(r.Method) + " " + routePattern(r)
			}),
		)
		// Skipped paths bypass otelhttp and routeHandler entirely: no span,
		// no duration. (An otelhttp filter would still run routeHandler.)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, skip := cfg.skipPaths[r.URL.Path]; skip {
				next.ServeHTTP(w, r)
				return
			}
			// otelhttp's writer wrapper swallows flush errors and has no
			// Unwrap, so the original writer is handed to tracingWriter
			// through the context for FlushError.
			traced.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), origWriterKey{}, w)))
		})
	}
}

// origWriterKey carries the ResponseWriter Middleware received.
type origWriterKey struct{}

// routeHandler runs inside the otelhttp span. It finalises the span name and
// route once routing is done and records the duration histogram.
type routeHandler struct {
	next     http.Handler
	cfg      *middlewareConfig
	duration metric.Float64Histogram
}

func (h *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Name, route and the hijack status go through the filtering wrapper,
	// like otelhttp's own writes; the handler sees the raw span.
	var span trace.Span = trace.SpanFromContext(r.Context())
	if ss, ok := r.Context().Value(serverSpanKey{}).(*serverSpan); ok {
		span = ss
	}
	orig, _ := r.Context().Value(origWriterKey{}).(http.ResponseWriter)
	tw := &tracingWriter{ResponseWriter: w, orig: orig, span: span, req: r, status: http.StatusOK}

	h.next.ServeHTTP(tw, r)

	route := tw.finish()
	if isWebSocketUpgrade(r) || tw.hijacked {
		return
	}
	if _, excluded := h.cfg.excludeDuration[route]; excluded {
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	h.duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributeSet(attribute.NewSet(
		attribute.String("http.request.method", metricMethod(r.Method)),
		attribute.String("url.scheme", scheme),
		attribute.Int("http.response.status_code", tw.status),
		attribute.String("http.route", route),
	)))
}

// routePattern returns the matched chi route pattern or UnmatchedRoute.
func routePattern(r *http.Request) string { return routePatternCtx(r.Context()) }

// routePatternCtx returns the matched chi route pattern in ctx or
// UnmatchedRoute.
func routePatternCtx(ctx context.Context) string {
	if rctx := chi.RouteContext(ctx); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return UnmatchedRoute
}

// tracingWriter captures the status code and ends the span when the
// connection is hijacked.
type tracingWriter struct {
	http.ResponseWriter
	// orig is the writer otelhttp wrapped, used to flush so that a flush
	// error is reported (otelhttp's wrapper discards it). nil: flush through
	// ResponseWriter.
	orig     http.ResponseWriter
	span     trace.Span
	req      *http.Request
	status   int
	wrote    bool
	hijacked bool
	route    string
}

func (w *tracingWriter) WriteHeader(code int) {
	if !w.wrote && code >= 200 {
		w.status = code
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *tracingWriter) Write(b []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *tracingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Flush flushes the underlying writer when it supports flushing.
func (w *tracingWriter) Flush() { _ = w.FlushError() }

// FlushError flushes the underlying writer and returns its error, so
// http.ResponseController.Flush reports failures through this wrapper.
func (w *tracingWriter) FlushError() error {
	if w.orig == nil {
		w.wrote = true
		return http.NewResponseController(w.ResponseWriter).Flush()
	}
	// A flush commits the response. Write the implicit 200 through the
	// otelhttp wrapper so its status bookkeeping matches, then flush the
	// original writer directly.
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	w.wrote = true
	return http.NewResponseController(w.orig).Flush()
}

// Hijack hands the connection to the handler and ends the span: from here on
// the connection is no longer an HTTP request.
func (w *tracingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	w.hijacked = true
	w.status = http.StatusSwitchingProtocols
	w.finish()
	w.span.SetAttributes(attribute.Int("http.response.status_code", http.StatusSwitchingProtocols))
	w.span.End()
	return conn, rw, nil
}

// finish names the span after the matched route. It runs once.
func (w *tracingWriter) finish() string {
	if w.route != "" {
		return w.route
	}
	w.route = routePattern(w.req)
	w.span.SetName(spanMethod(w.req.Method) + " " + w.route)
	w.span.SetAttributes(attribute.String("http.route", w.route))
	return w.route
}

// isWebSocketUpgrade reports whether r asks to switch to the WebSocket
// protocol (Connection: Upgrade and Upgrade: websocket).
func isWebSocketUpgrade(r *http.Request) bool {
	if !headerHasToken(r.Header, "Upgrade", "websocket") {
		return false
	}
	return headerHasToken(r.Header, "Connection", "upgrade")
}

func headerHasToken(h http.Header, name, token string) bool {
	for _, v := range h.Values(name) {
		for _, t := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(t), token) {
				return true
			}
		}
	}
	return false
}

var knownMethods = map[string]struct{}{
	http.MethodConnect: {}, http.MethodDelete: {}, http.MethodGet: {}, http.MethodHead: {},
	http.MethodOptions: {}, http.MethodPatch: {}, http.MethodPost: {}, http.MethodPut: {},
	http.MethodTrace: {},
}

// spanMethod follows the span-name convention: unknown methods become "HTTP".
func spanMethod(m string) string {
	if _, ok := knownMethods[m]; ok {
		return m
	}
	return "HTTP"
}

// metricMethod follows the attribute convention: unknown methods become
// "_OTHER", which bounds the label's cardinality.
func metricMethod(m string) string {
	if _, ok := knownMethods[m]; ok {
		return m
	}
	return "_OTHER"
}

// scrubbingTracerProvider hands out tracers whose spans keep only the
// serverSpanAttributes allowlist. otelhttp sets some attributes at span start
// and others (the response attributes) after the handler returns, so
// filtering happens both on the start options and on the span itself.
type scrubbingTracerProvider struct {
	embedded.TracerProvider
	tp trace.TracerProvider // nil: resolve the global on each Start
}

func (p scrubbingTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return scrubbingTracer{name: name, opts: opts, tp: p.tp}
}

type scrubbingTracer struct {
	embedded.Tracer
	name string
	opts []trace.TracerOption
	tp   trace.TracerProvider
}

func (t scrubbingTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tp := t.tp
	if tp == nil {
		tp = otelapi.GetTracerProvider()
	}
	// spanName comes from Middleware's formatter: "{METHOD}" or
	// "{METHOD} {pattern}". Only the method token is kept; the name is
	// rebuilt from the chi route on every SetName.
	method, _, _ := strings.Cut(spanName, " ")
	ss := &serverSpan{ctx: ctx, method: spanMethod(method)}

	cfg := trace.NewSpanStartConfig(opts...)
	clean := []trace.SpanStartOption{
		trace.WithAttributes(ss.allowed(cfg.Attributes())...),
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
	ctx, span := tp.Tracer(t.name, t.opts...).Start(ctx, ss.method, clean...)
	ss.Span = span
	// otelhttp writes through the returned serverSpan, so its request-derived
	// attributes are filtered. The context it passes down the chain carries
	// the raw span: what the application itself records on the server span
	// (its own attributes, errors, events) is the application's choice and
	// is kept. routeHandler finds the filtering wrapper under serverSpanKey.
	ctx = context.WithValue(trace.ContextWithSpan(ctx, span), serverSpanKey{}, ss)
	return ctx, ss
}

// serverSpanKey carries the *serverSpan for routeHandler.
type serverSpanKey struct{}

// serverSpan filters what otelhttp (and this package) write to a server
// span. Application code gets the raw span from the request context.
type serverSpan struct {
	trace.Span
	// ctx is the request context the span was started with. chi's route
	// context in it is filled in during routing, so the matched pattern can
	// be read from it later.
	ctx    context.Context
	method string // normalised by spanMethod
}

// allowed keeps the serverSpanAttributes allowlist and normalises values
// that would otherwise come from the request: the method is bounded, the
// scheme must be http or https, and http.route is always the chi pattern.
func (s *serverSpan) allowed(in []attribute.KeyValue) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(in))
	for _, kv := range in {
		if _, ok := serverSpanAttributes[kv.Key]; !ok {
			continue
		}
		switch kv.Key {
		case "http.request.method":
			kv = attribute.String("http.request.method", metricMethod(kv.Value.AsString()))
		case "url.scheme":
			if v := kv.Value.AsString(); v != "http" && v != "https" {
				continue
			}
		case "http.route":
			kv = attribute.String("http.route", routePatternCtx(s.ctx))
		}
		out = append(out, kv)
	}
	return out
}

func (s *serverSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.Span.SetAttributes(s.allowed(kv)...)
}

// SetName ignores name and uses "{METHOD} {route}", built from the matched
// chi route pattern (or UnmatchedRoute), so the raw path can never become
// the span name.
func (s *serverSpan) SetName(string) {
	s.Span.SetName(s.method + " " + routePatternCtx(s.ctx))
}

func (s *serverSpan) SetStatus(code codes.Code, description string) {
	if description != "" {
		description = serverErrorDescription
	}
	s.Span.SetStatus(code, description)
}

// RecordError is dropped: error text may quote request data. A failure is
// still visible through the status and error.type.
func (s *serverSpan) RecordError(error, ...trace.EventOption) {}

// AddEvent is dropped: otelhttp events carry no information the allowlist
// keeps, and a future event could carry request details.
func (s *serverSpan) AddEvent(string, ...trace.EventOption) {}
