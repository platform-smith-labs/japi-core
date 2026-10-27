package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Log attribute keys added by TraceHandler. They follow the OpenTelemetry
// log data model field names so log pipelines can join log lines to traces.
const (
	LogKeyTraceID    = "trace_id"
	LogKeySpanID     = "span_id"
	LogKeyTraceFlags = "trace_flags"
)

// LogConfig configures TraceHandler. The zero value adds trace correlation
// and filters nothing.
type LogConfig struct {
	// ForbiddenKeys are attribute keys that are always dropped. A key matches
	// the attribute's own (leaf) key at any depth: "email" drops a top-level
	// "email" attribute, one added through Logger.With, and one nested inside
	// a group such as "user". Dotted paths ("user.email") are not interpreted.
	// A group attribute whose own key is forbidden is dropped with everything
	// inside it.
	ForbiddenKeys []string

	// Redact, when non-nil, is applied to every non-group attribute that
	// survives ForbiddenKeys, with the same contract as
	// slog.HandlerOptions.ReplaceAttr: groups lists the enclosing group names
	// (from Logger.WithGroup and group attributes), the attribute's value is
	// already resolved, and returning a zero slog.Attr drops the attribute.
	// It is not called for group attributes themselves (it is called for
	// their members), nor for the trace keys TraceHandler adds or the
	// record's built-in time/level/message fields.
	//
	// Redact may be called concurrently and must not retain groups.
	Redact func(groups []string, a slog.Attr) slog.Attr
}

// TraceHandler wraps next so that:
//
//   - every record logged with a context carrying a valid span context gets
//     trace_id and span_id (lowercase hex) and trace_flags (two hex digits,
//     e.g. "01" when sampled) as top-level attributes, even when the logger
//     has open groups;
//   - when it adds them, top-level attributes the caller passed under the
//     same three keys (with the record or through Logger.With) are dropped,
//     so a line never carries a key twice. Without a span in the context,
//     such caller attributes are kept (Logger.With ones are emitted at the
//     top level, after the record's own attributes);
//   - attributes named in cfg.ForbiddenKeys are dropped and cfg.Redact is
//     applied, both to attributes passed with the record and to those added
//     through Logger.With / Handler.WithAttrs, including members of groups.
//
// Trace keys are only added when a context is passed to the logger
// (InfoContext, ErrorContext, Log, ...); Info and friends use
// context.Background and get none.
//
// The returned handler is immutable and safe for concurrent use as long as
// next is.
//
// Example:
//
//	logger := slog.New(otel.TraceHandler(
//	    slog.NewJSONHandler(os.Stdout, nil),
//	    otel.LogConfig{ForbiddenKeys: []string{"password", "authorization"}},
//	))
//	logger.InfoContext(r.Context(), "order placed", "order_id", id)
func TraceHandler(next slog.Handler, cfg LogConfig) slog.Handler {
	f := &logFilter{redact: cfg.Redact}
	if len(cfg.ForbiddenKeys) > 0 {
		f.forbidden = make(map[string]struct{}, len(cfg.ForbiddenKeys))
		for _, k := range cfg.ForbiddenKeys {
			f.forbidden[k] = struct{}{}
		}
	}
	return &traceHandler{next: next, root: next, filter: f}
}

// logFilter holds the immutable filtering configuration shared by every
// handler derived from one TraceHandler call.
type logFilter struct {
	forbidden map[string]struct{}
	redact    func(groups []string, a slog.Attr) slog.Attr
}

func (f *logFilter) active() bool { return f.forbidden != nil || f.redact != nil }

// isTraceKey reports whether key is one of the keys TraceHandler adds.
func isTraceKey(key string) bool {
	return key == LogKeyTraceID || key == LogKeySpanID || key == LogKeyTraceFlags
}

// hasTopLevelTraceKey reports whether attrs, logged with no open group,
// contain a trace key at the top level. Inline groups (empty key) put their
// members at the top level too.
func hasTopLevelTraceKey(attrs []slog.Attr) bool {
	for _, a := range attrs {
		if isTraceKey(a.Key) {
			return true
		}
		if a.Key == "" {
			if v := a.Value.Resolve(); v.Kind() == slog.KindGroup && hasTopLevelTraceKey(v.Group()) {
				return true
			}
		}
	}
	return false
}

// handlerOp is one WithAttrs or WithGroup call made after the first group was
// opened. Ops are replayed on root when trace keys must be placed outside the
// open groups.
type handlerOp struct {
	group string      // WithGroup when attrs is nil
	attrs []slog.Attr // WithAttrs (already filtered)
}

type traceHandler struct {
	// next has every WithAttrs/WithGroup applied; it handles records that
	// need no top-level trace keys.
	next slog.Handler
	// root has the WithAttrs calls made before the first WithGroup applied.
	root slog.Handler
	// ops are the calls made since the first WithGroup, in order.
	ops []handlerOp
	// groups are the currently open group names, for Redact.
	groups []string
	// bound are trace-key attributes added through WithAttrs at the top
	// level. They are held back and emitted only when the record's context
	// has no span, so they never duplicate the keys this handler adds.
	bound  []slog.Attr
	filter *logFilter
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	attrs = h.filter.attrs(h.groups, attrs)
	var bound []slog.Attr
	if len(h.groups) == 0 && hasTopLevelTraceKey(attrs) {
		attrs, bound = splitTraceKeys(attrs)
	}
	if len(attrs) == 0 && len(bound) == 0 {
		return h
	}
	c := *h
	if len(bound) > 0 {
		c.bound = mergeBound(h.bound, bound)
	}
	if len(attrs) == 0 {
		return &c
	}
	c.next = h.next.WithAttrs(attrs)
	if len(h.groups) == 0 {
		c.root = c.next
	} else {
		c.ops = appendOp(h.ops, handlerOp{attrs: attrs})
	}
	return &c
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.next = h.next.WithGroup(name)
	c.ops = appendOp(h.ops, handlerOp{group: name})
	c.groups = append(h.groups[:len(h.groups):len(h.groups)], name)
	return &c
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	var sc trace.SpanContext
	if ctx != nil {
		sc = trace.SpanContextFromContext(ctx)
	}
	// Caller-supplied top-level trace keys give way to the handler's own.
	dropTrace := sc.IsValid() && len(h.groups) == 0 && recordHasTraceKey(r)

	if h.filter.active() || dropTrace {
		nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		r.Attrs(func(a slog.Attr) bool {
			if dropTrace {
				kept, _ := splitTraceKeys([]slog.Attr{a})
				if len(kept) == 0 {
					return true
				}
				a = kept[0]
			}
			if fa, ok := h.filter.attr(h.groups, a); ok {
				nr.AddAttrs(fa)
			}
			return true
		})
		r = nr
	}

	traceAttrs := h.bound
	if sc.IsValid() {
		traceAttrs = []slog.Attr{
			slog.String(LogKeyTraceID, sc.TraceID().String()),
			slog.String(LogKeySpanID, sc.SpanID().String()),
			slog.String(LogKeyTraceFlags, sc.TraceFlags().String()),
		}
	}
	if len(traceAttrs) == 0 {
		return h.next.Handle(ctx, r)
	}

	if len(h.groups) == 0 {
		r = r.Clone()
		r.AddAttrs(traceAttrs...)
		return h.next.Handle(ctx, r)
	}

	// Groups are open: record attributes would land inside them. Rebuild the
	// handler with the trace keys at the top level, then replay the grouped
	// calls. This costs a few allocations and only happens for grouped
	// loggers that log with a span in context.
	nh := h.root.WithAttrs(traceAttrs)
	for _, op := range h.ops {
		if op.attrs != nil {
			nh = nh.WithAttrs(op.attrs)
		} else {
			nh = nh.WithGroup(op.group)
		}
	}
	return nh.Handle(ctx, r)
}

// splitTraceKeys separates top-level trace-key attributes (including those
// inside inline groups, which are emitted at the top level) from the rest.
func splitTraceKeys(attrs []slog.Attr) (rest, traceKeys []slog.Attr) {
	for _, a := range attrs {
		switch {
		case isTraceKey(a.Key):
			traceKeys = append(traceKeys, a)
		case a.Key == "" && a.Value.Resolve().Kind() == slog.KindGroup:
			r, tk := splitTraceKeys(a.Value.Resolve().Group())
			traceKeys = append(traceKeys, tk...)
			if len(r) > 0 {
				rest = append(rest, slog.Attr{Value: slog.GroupValue(r...)})
			}
		default:
			rest = append(rest, a)
		}
	}
	return rest, traceKeys
}

// mergeBound returns old with add applied, a later value replacing an
// earlier one with the same key. It never writes into old's backing array.
func mergeBound(old, add []slog.Attr) []slog.Attr {
	out := make([]slog.Attr, 0, len(old)+len(add))
	for _, a := range old {
		if !containsKey(add, a.Key) {
			out = append(out, a)
		}
	}
	for i, a := range add {
		if !containsKey(add[i+1:], a.Key) {
			out = append(out, a)
		}
	}
	return out
}

func containsKey(attrs []slog.Attr, key string) bool {
	for _, a := range attrs {
		if a.Key == key {
			return true
		}
	}
	return false
}

// recordHasTraceKey reports whether r carries a top-level trace key.
func recordHasTraceKey(r slog.Record) bool {
	found := false
	r.Attrs(func(a slog.Attr) bool {
		found = hasTopLevelTraceKey([]slog.Attr{a})
		return !found
	})
	return found
}

// appendOp returns ops with op appended, never writing into ops' backing
// array, so handlers derived from the same parent do not share state.
func appendOp(ops []handlerOp, op handlerOp) []handlerOp {
	out := make([]handlerOp, len(ops), len(ops)+1)
	copy(out, ops)
	return append(out, op)
}

// attrs filters a list, returning nil when nothing survives.
func (f *logFilter) attrs(groups []string, in []slog.Attr) []slog.Attr {
	if !f.active() {
		return in
	}
	out := make([]slog.Attr, 0, len(in))
	for _, a := range in {
		if fa, ok := f.attr(groups, a); ok {
			out = append(out, fa)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// attr applies ForbiddenKeys then Redact to a, recursing into groups. ok is
// false when the attribute is dropped.
func (f *logFilter) attr(groups []string, a slog.Attr) (slog.Attr, bool) {
	if _, bad := f.forbidden[a.Key]; bad {
		return slog.Attr{}, false
	}
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		inner := groups
		if a.Key != "" {
			inner = append(groups[:len(groups):len(groups)], a.Key)
		}
		members := f.attrs(inner, a.Value.Group())
		if len(members) == 0 {
			return slog.Attr{}, false
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(members...)}, true
	}
	if f.redact != nil {
		a = f.redact(groups, a)
		if a.Equal(slog.Attr{}) {
			return slog.Attr{}, false
		}
	}
	return a, true
}
