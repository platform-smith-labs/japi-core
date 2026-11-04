// Package spanstatus lets code inside a request mark the server span as
// failed with a description that later instrumentation cannot overwrite.
//
// otelhttp sets the server span status from the response code after the
// handler returns. For a 5xx that is (Error, ""), which would replace a more
// specific description such as "panic" or an error code. The otel package's
// server span implements Pinner and is stored in the request context; Pin
// reaches it from any depth of the handler chain.
package spanstatus

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Pinner is implemented by a server span that can hold a final error status.
type Pinner interface {
	// Pin sets the span status to Error with description and error.type to
	// errorType, and ignores later status and error.type writes made by the
	// HTTP instrumentation. Both values must be bounded constants (an error
	// code, "panic"), never request data or error text.
	Pin(description, errorType string)
}

type key struct{}

// With returns ctx carrying p.
func With(ctx context.Context, p Pinner) context.Context {
	return context.WithValue(ctx, key{}, p)
}

// Pin marks the request's server span as failed. When ctx carries no Pinner
// (no tracing middleware) it sets the status on the current span instead.
func Pin(ctx context.Context, description, errorType string) {
	if p, ok := ctx.Value(key{}).(Pinner); ok {
		p.Pin(description, errorType)
		return
	}
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Error, description)
	span.SetAttributes(attribute.String("error.type", errorType))
}
