package otel

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	httpmiddleware "github.com/platform-smith-labs/japi-core/v4/middleware/http"
)

// maxInboundRequestIDLen bounds the inbound X-Request-ID recorded on a span.
const maxInboundRequestIDLen = 128

// RequestIDFromTrace makes the request's trace ID its request ID: it sets
// the X-Request-ID response header to the trace ID and stores it as the
// request ID (httpmiddleware.GetRequestID returns it), so the value a client
// quotes finds the trace, the log lines and the trace_id of an error body.
//
// An inbound X-Request-ID never becomes the trace ID. When the request
// carries no traceparent, the inbound value is recorded on the current span
// as http.request.header.x-request-id (printable ASCII, at most 128
// characters) so a caller's own ID can still be searched.
//
// Install it inside the tracing middleware. Without a valid span (no SDK
// installed) it does nothing and any request ID set earlier stays.
func RequestIDFromTrace() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id := TraceID(ctx)
			if id == "" {
				next.ServeHTTP(w, r)
				return
			}
			if inbound := r.Header.Get(httpmiddleware.RequestIDHeader); inbound != "" && r.Header.Get("traceparent") == "" {
				if v := sanitizeRequestID(inbound); v != "" {
					trace.SpanFromContext(ctx).SetAttributes(
						attribute.StringSlice("http.request.header.x-request-id", []string{v}))
				}
			}
			w.Header().Set(httpmiddleware.RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(httpmiddleware.ContextWithRequestID(ctx, id)))
		})
	}
}

// sanitizeRequestID keeps printable ASCII and truncates.
func sanitizeRequestID(s string) string {
	out := make([]byte, 0, min(len(s), maxInboundRequestIDLen))
	for i := 0; i < len(s) && len(out) < maxInboundRequestIDLen; i++ {
		if c := s[i]; c >= 0x20 && c < 0x7f {
			out = append(out, c)
		}
	}
	return string(out)
}
