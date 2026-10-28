package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/platform-smith-labs/japi-core/v4/internal/spanstatus"
)

// ProblemContentType is the media type of every error body.
const ProblemContentType = "application/problem+json"

// WriteOption configures WriteProblem and WriteError.
type WriteOption func(*writeConfig)

type writeConfig struct {
	logger   *slog.Logger
	logAttrs []slog.Attr
	recorded bool
}

// WithLogger sets the logger for the error line. Pass the request-scoped
// logger so the line carries its fields. Default: slog.Default(). The line is
// always logged with the request context, so a handler that adds trace or
// identity fields from the context sees it.
func WithLogger(l *slog.Logger) WriteOption {
	return func(c *writeConfig) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithLogAttrs adds attributes to the error line.
func WithLogAttrs(attrs ...slog.Attr) WriteOption {
	return func(c *writeConfig) { c.logAttrs = append(c.logAttrs, attrs...) }
}

// AlreadyRecorded tells the writer that the caller has already logged the
// failure and recorded it on the span (recovery middleware does), so the
// writer only writes the body.
func AlreadyRecorded() WriteOption {
	return func(c *writeConfig) { c.recorded = true }
}

// WriteError writes err as a Problem Details response.
//
// A *Problem found with errors.As is written as constructed. Any other error is written as the generic 500 internal_error problem; its text
// goes to the log and the span, never to the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error, opts ...WriteOption) error {
	if err == nil {
		err = errors.New("nil error passed to WriteError")
	}
	var p *Problem
	if errors.As(err, &p) && p != nil {
		if _, status := Classify(p); err != error(p) && status >= 500 {
			// The problem was wrapped: keep the outer context for the log of
			// a server error. A 4xx wrapper's text can quote the request.
			opts = append(opts, WithLogAttrs(slog.String("error.wrapped", err.Error())))
		}
		return WriteProblem(w, r, p, opts...)
	}
	return WriteProblem(w, r, Internal(err), opts...)
}

// resolved is a problem after registry resolution, ready to write.
type resolved struct {
	code       Code
	status     int
	typ        string
	title      string
	detail     string // written only for status < 500
	errors     []FieldError
	extensions map[string]any
	logAttrs   []slog.Attr // registry corrections, for the log line
}

// resolve applies the registry and the wire rules to p.
func resolve(p *Problem) resolved {
	out := resolved{code: p.Code, status: p.Status, detail: p.Detail, errors: p.Errors}

	internal := func(reason string) {
		out.logAttrs = append(out.logAttrs,
			slog.String("error.rejected_code", string(out.code)),
			slog.Int("error.rejected_status", out.status),
			slog.String("error.rejected_reason", reason))
		out.code = CodeInternalError
		out.status = http.StatusInternalServerError
		out.errors = nil
	}

	if !validCode(out.code) {
		violation("code %q is not a lowercase snake_case token", out.code)
		internal("invalid code")
	}
	info, found, registered := lookupCode(out.code)
	if registered && !found {
		violation("code %q is not registered", out.code)
		internal("unregistered code")
		info, found, _ = lookupCode(out.code)
	}

	if out.status == 0 {
		if found && info.Status != 0 {
			out.status = info.Status
		} else {
			out.status = http.StatusInternalServerError
		}
	}
	if out.status < 400 || out.status > 599 {
		violation("code %q constructed with non-error status %d", out.code, out.status)
		internal("non-error status")
		info, found, _ = lookupCode(out.code)
	}
	if found && info.Status != 0 && out.status != info.Status {
		violation("code %q constructed with status %d, registered with %d", out.code, out.status, info.Status)
		out.logAttrs = append(out.logAttrs, slog.Int("error.registered_status", info.Status))
	}

	base := currentTypeBase()
	if base == "" {
		out.typ = "about:blank"
		out.title = http.StatusText(out.status)
	} else {
		out.typ = base + string(out.code)
		out.title = info.Title
		if out.title == "" {
			out.title = http.StatusText(out.status)
		}
	}

	if len(p.Extensions) > 0 {
		for _, k := range sortedKeys(p.Extensions) {
			if _, std := standardMembers[k]; std {
				violation("extension %q would replace a standard member", k)
				out.logAttrs = append(out.logAttrs, slog.String("error.dropped_extension", k))
				continue
			}
			if registered && !(found && slices.Contains(info.Extensions, k)) {
				violation("extension %q is not declared for code %q", k, out.code)
				out.logAttrs = append(out.logAttrs, slog.String("error.dropped_extension", k))
				continue
			}
			if out.extensions == nil {
				out.extensions = make(map[string]any, len(p.Extensions))
			}
			out.extensions[k] = p.Extensions[k]
		}
	}
	return out
}

// body returns the JSON members for res, with traceID when non-empty.
func (res resolved) body(traceID string) map[string]any {
	b := make(map[string]any, 7+len(res.extensions))
	for k, v := range res.extensions {
		b[k] = v
	}
	b["type"] = res.typ
	b["title"] = res.title
	b["status"] = res.status
	if res.status < 500 {
		if res.detail != "" {
			b["detail"] = res.detail
		}
		if len(res.errors) > 0 {
			b["errors"] = res.errors
		}
	}
	if traceID != "" {
		b["trace_id"] = traceID
	}
	return b
}

// traceIDFrom returns the hex trace ID of the span in ctx, or "".
func traceIDFrom(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}

// WriteProblem writes p as an RFC 9457 Problem Details response, logs one
// line through the request context, and records the failure on the span.
// It works with any net/http handler; no router is required.
//
// The body always carries "trace_id" when the request has a span. For a 5xx
// the client gets only type, title, status, trace_id and declared
// extensions: Detail, Errors and Cause go to the log line, and Cause is also
// recorded on the span (exception event, status Error with the code as its
// description).
func WriteProblem(w http.ResponseWriter, r *http.Request, p *Problem, opts ...WriteOption) error {
	cfg := writeConfig{logger: slog.Default()}
	for _, o := range opts {
		o(&cfg)
	}
	if p == nil {
		p = Internal(errors.New("nil problem passed to WriteProblem"))
	}
	ctx := r.Context()
	res := resolve(p)
	traceID := traceIDFrom(ctx)

	if !cfg.recorded {
		recordProblem(ctx, p, res)
		logProblem(ctx, cfg, r, p, res)
	}

	payload, err := json.Marshal(res.body(traceID))
	if err != nil {
		// An extension value that cannot be encoded: drop the extensions
		// rather than fail the error response itself.
		cfg.logger.LogAttrs(ctx, slog.LevelError, "problem extensions could not be encoded",
			slog.String("error.type", string(res.code)))
		res.extensions = nil
		payload, err = json.Marshal(res.body(traceID))
		if err != nil {
			return err
		}
	}
	w.Header().Set("Content-Type", ProblemContentType)
	w.Header().Del("Content-Length")
	w.WriteHeader(res.status)
	_, err = w.Write(append(payload, '\n'))
	return err
}

// recordProblem sets error.type on the current span and, for a 5xx, records
// the cause and pins the server span's status.
func recordProblem(ctx context.Context, p *Problem, res resolved) {
	span := trace.SpanFromContext(ctx)
	if res.status < 500 {
		span.SetAttributes(attribute.String("error.type", string(res.code)))
		return
	}
	if p.Cause != nil {
		span.RecordError(p.Cause)
	}
	spanstatus.Pin(ctx, string(res.code), string(res.code))
}

// logProblem writes the one log line for an error response.
func logProblem(ctx context.Context, cfg writeConfig, r *http.Request, p *Problem, res resolved) {
	level := slog.LevelWarn
	if res.status >= 500 {
		level = slog.LevelError
	}
	attrs := []slog.Attr{
		slog.Int("http.response.status_code", res.status),
		slog.String("error.type", string(res.code)),
		slog.String("http.request.method", r.Method),
		slog.String("http.route", requestRoute(r)),
	}
	if p.Cause != nil {
		// Only the cause's Go type is logged for a 4xx: a client error's
		// cause comes from parsing the request (a JSON fragment, a CSV
		// cell, a parameter) and its text can quote customer content. A
		// server error's cause is the operator's diagnosis and is logged.
		attrs = append(attrs, slog.String("error.cause_type", causeType(p.Cause)))
		if res.status >= 500 {
			attrs = append(attrs, slog.String("error", p.Cause.Error()))
		}
	}
	if res.status >= 500 {
		// Not on the wire for a 5xx; kept for the operator.
		if p.Detail != "" {
			attrs = append(attrs, slog.String("error.detail", p.Detail))
		}
		if len(p.Errors) > 0 {
			attrs = append(attrs, slog.Int("error.field_count", len(p.Errors)))
		}
	}
	attrs = append(attrs, res.logAttrs...)
	attrs = append(attrs, cfg.logAttrs...)
	cfg.logger.LogAttrs(ctx, level, "error response", attrs...)
}

// causeType returns the Go type of the innermost error in err's Unwrap
// chain (following the first branch of a joined error), e.g. "*json.SyntaxError".
func causeType(err error) string {
	for {
		var next error
		switch u := err.(type) {
		case interface{ Unwrap() error }:
			next = u.Unwrap()
		case interface{ Unwrap() []error }:
			if errs := u.Unwrap(); len(errs) > 0 {
				next = errs[0]
			}
		}
		if next == nil {
			return fmt.Sprintf("%T", err)
		}
		err = next
	}
}

// Classify returns the code and status WriteError would use for err, for
// log lines that must describe an error without quoting its text.
func Classify(err error) (Code, int) {
	var p *Problem
	if errors.As(err, &p) && p != nil {
		status := p.Status
		if status == 0 {
			if info, found, _ := lookupCode(p.Code); found && info.Status != 0 {
				status = info.Status
			} else {
				status = http.StatusInternalServerError
			}
		}
		return p.Code, status
	}
	return CodeInternalError, http.StatusInternalServerError
}
