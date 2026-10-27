package otel

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// newLogCapture returns a logger writing JSON through TraceHandler into buf.
func newLogCapture(cfg LogConfig) (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(TraceHandler(slog.NewJSONHandler(buf, nil), cfg)), buf
}

// decodeLine decodes the single JSON log line in buf.
func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %q", len(lines), buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("decode %q: %v", lines[0], err)
	}
	return m
}

func spanContext(t *testing.T) (context.Context, trace.SpanContext) {
	t.Helper()
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })
	return ctx, span.SpanContext()
}

func TestTraceHandler_AddsTraceKeysFromSpan(t *testing.T) {
	ctx, sc := spanContext(t)
	logger, buf := newLogCapture(LogConfig{})

	logger.InfoContext(ctx, "hello", "k", "v")
	m := decodeLine(t, buf)

	if m["trace_id"] != sc.TraceID().String() {
		t.Errorf("trace_id = %v, want %s", m["trace_id"], sc.TraceID())
	}
	if m["span_id"] != sc.SpanID().String() {
		t.Errorf("span_id = %v, want %s", m["span_id"], sc.SpanID())
	}
	if m["trace_flags"] != "01" {
		t.Errorf("trace_flags = %v, want 01", m["trace_flags"])
	}
	if m["k"] != "v" {
		t.Errorf("record attr lost: %v", m)
	}
}

func TestTraceHandler_NoSpanNoTraceKeys(t *testing.T) {
	logger, buf := newLogCapture(LogConfig{})
	logger.InfoContext(context.Background(), "hello")
	logger.Info("no ctx")
	for _, k := range []string{"trace_id", "span_id", "trace_flags"} {
		if strings.Contains(buf.String(), k) {
			t.Errorf("unexpected %s in %q", k, buf.String())
		}
	}
}

func TestTraceHandler_TraceKeysStayTopLevelInsideGroups(t *testing.T) {
	ctx, sc := spanContext(t)
	logger, buf := newLogCapture(LogConfig{})

	logger.With("a", 1).WithGroup("g").With("b", 2).InfoContext(ctx, "hello", "c", 3)
	m := decodeLine(t, buf)

	if m["trace_id"] != sc.TraceID().String() || m["a"] != float64(1) {
		t.Fatalf("top level wrong: %v", m)
	}
	g, ok := m["g"].(map[string]any)
	if !ok || g["b"] != float64(2) || g["c"] != float64(3) {
		t.Fatalf("group wrong: %v", m)
	}
	if _, ok := g["trace_id"]; ok {
		t.Errorf("trace_id nested in group: %v", g)
	}
}

func TestTraceHandler_ForbiddenKeys(t *testing.T) {
	cfg := LogConfig{ForbiddenKeys: []string{"secret", "creds"}}

	t.Run("top level", func(t *testing.T) {
		logger, buf := newLogCapture(cfg)
		logger.Info("m", "secret", "TOP-MARKER", "ok", 1)
		assertDropped(t, buf, "TOP-MARKER", "ok")
	})
	t.Run("WithAttrs", func(t *testing.T) {
		logger, buf := newLogCapture(cfg)
		logger.With("secret", "WITH-MARKER", "ok", 1).Info("m")
		assertDropped(t, buf, "WITH-MARKER", "ok")
	})
	t.Run("group attr", func(t *testing.T) {
		logger, buf := newLogCapture(cfg)
		logger.Info("m", slog.Group("user", "secret", "GROUP-MARKER", "ok", 1))
		assertDropped(t, buf, "GROUP-MARKER", "ok")
	})
	t.Run("WithGroup then WithAttrs", func(t *testing.T) {
		logger, buf := newLogCapture(cfg)
		logger.WithGroup("g").With("secret", "WG-MARKER", "ok", 1).Info("m")
		assertDropped(t, buf, "WG-MARKER", "ok")
	})
	t.Run("forbidden group key drops the group", func(t *testing.T) {
		logger, buf := newLogCapture(cfg)
		logger.Info("m", slog.Group("creds", "user", "CREDS-MARKER"), "ok", 1)
		assertDropped(t, buf, "CREDS-MARKER", "ok")
	})
	t.Run("LogValuer is resolved before filtering", func(t *testing.T) {
		logger, buf := newLogCapture(cfg)
		logger.Info("m", "v", groupValuer{}, "ok", 1)
		assertDropped(t, buf, "VALUER-MARKER", "ok")
	})
}

type groupValuer struct{}

func (groupValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String("secret", "VALUER-MARKER"))
}

func assertDropped(t *testing.T, buf *bytes.Buffer, marker, keep string) {
	t.Helper()
	out := buf.String()
	if strings.Contains(out, marker) {
		t.Errorf("forbidden value %q logged: %s", marker, out)
	}
	if !strings.Contains(out, `"`+keep+`"`) {
		t.Errorf("allowed key %q missing: %s", keep, out)
	}
}

func TestTraceHandler_Redact(t *testing.T) {
	var mu sync.Mutex
	var seenGroups [][]string
	cfg := LogConfig{
		ForbiddenKeys: []string{"dropped"},
		Redact: func(groups []string, a slog.Attr) slog.Attr {
			mu.Lock()
			seenGroups = append(seenGroups, append([]string(nil), groups...))
			mu.Unlock()
			switch a.Key {
			case "email":
				return slog.String(a.Key, "[redacted]")
			case "token":
				return slog.Attr{} // drop
			case "dropped":
				t.Error("Redact called for a forbidden key")
			}
			return a
		},
	}
	logger, buf := newLogCapture(cfg)

	logger.With("email", "a@example.com").WithGroup("g").
		Info("m", "token", "TOKEN-MARKER", slog.Group("inner", "email", "b@example.com"), "dropped", 1, "ok", 1)
	out := buf.String()

	for _, leak := range []string{"a@example.com", "b@example.com", "TOKEN-MARKER", `"token"`, `"dropped"`} {
		if strings.Contains(out, leak) {
			t.Errorf("%s leaked: %s", leak, out)
		}
	}
	if strings.Count(out, "[redacted]") != 2 {
		t.Errorf("want 2 redacted values: %s", out)
	}
	m := decodeLine(t, buf)
	inner := m["g"].(map[string]any)["inner"].(map[string]any)
	if inner["email"] != "[redacted]" {
		t.Errorf("nested redaction wrong: %v", m)
	}

	want := map[string]bool{"": false, "g": false, "g/inner": false}
	for _, g := range seenGroups {
		want[strings.Join(g, "/")] = true
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("Redact never saw groups %q (saw %v)", k, seenGroups)
		}
	}
}

func TestTraceHandler_Enabled(t *testing.T) {
	h := TraceHandler(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn}), LogConfig{})
	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Info enabled under a Warn handler")
	}
	if !h.WithGroup("g").WithAttrs([]slog.Attr{slog.Int("a", 1)}).Enabled(context.Background(), slog.LevelError) {
		t.Error("Error disabled under a Warn handler")
	}
}

func TestTraceHandler_ConcurrentDerivedLoggers(t *testing.T) {
	ctx, _ := spanContext(t)
	var mu sync.Mutex
	buf := &bytes.Buffer{}
	base := slog.New(TraceHandler(slog.NewJSONHandler(lockedWriter{&mu, buf}, nil),
		LogConfig{ForbiddenKeys: []string{"secret"}}))
	parent := base.WithGroup("g")

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := parent.With("i", i, "secret", "RACE-MARKER")
			l.InfoContext(ctx, "m", "secret", "RACE-MARKER")
		}()
	}
	wg.Wait()
	if strings.Contains(buf.String(), "RACE-MARKER") {
		t.Error("forbidden value logged under concurrency")
	}
	if got := strings.Count(buf.String(), `"trace_id"`); got != 16 {
		t.Errorf("trace_id count = %d, want 16", got)
	}
}

type lockedWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

// countKey counts occurrences of "key": in a raw JSON line (a decoded map
// would hide duplicates).
func countKey(line, key string) int { return strings.Count(line, `"`+key+`":`) }

func TestTraceHandler_CallerTraceKeysDoNotDuplicate(t *testing.T) {
	ctx, sc := spanContext(t)
	logger, buf := newLogCapture(LogConfig{})

	logger.With("trace_id", "with-value", slog.Group("", "span_id", "inline-value")).
		InfoContext(ctx, "hello", "trace_flags", "record-value", "span_id", "record-value", "k", "v")
	line := buf.String()
	for _, k := range []string{LogKeyTraceID, LogKeySpanID, LogKeyTraceFlags} {
		if n := countKey(line, k); n != 1 {
			t.Errorf("%s appears %d times, want 1: %s", k, n, line)
		}
	}
	m := decodeLine(t, buf)
	if m["trace_id"] != sc.TraceID().String() || m["span_id"] != sc.SpanID().String() || m["trace_flags"] != "01" {
		t.Errorf("trace keys not the span's: %v", m)
	}
	if m["k"] != "v" {
		t.Errorf("other attribute lost: %v", m)
	}
}

func TestTraceHandler_CallerTraceKeysInsideGroupsAndWithoutSpan(t *testing.T) {
	ctx, sc := spanContext(t)

	// Inside a group the key is "g.trace_id", not a duplicate: kept.
	logger, buf := newLogCapture(LogConfig{})
	logger.WithGroup("g").InfoContext(ctx, "hello", "trace_id", "nested")
	line := buf.String()
	if countKey(line, LogKeyTraceID) != 2 || !strings.Contains(line, sc.TraceID().String()) {
		t.Errorf("want top-level and nested trace_id: %s", line)
	}

	// Without a span nothing is added, so caller keys are kept once, even
	// with a group open after Logger.With.
	logger, buf = newLogCapture(LogConfig{})
	logger.With("trace_id", "caller").WithGroup("g").InfoContext(context.Background(), "hello", "k", "v")
	line = buf.String()
	if countKey(line, LogKeyTraceID) != 1 {
		t.Errorf("caller trace_id count != 1: %s", line)
	}
	m := decodeLine(t, buf)
	if m["trace_id"] != "caller" {
		t.Errorf("trace_id = %v, want caller's value at top level", m["trace_id"])
	}

	// With a span and a group open after Logger.With: the span's keys win.
	logger, buf = newLogCapture(LogConfig{})
	logger.With("trace_id", "caller").WithGroup("g").InfoContext(ctx, "hello")
	line = buf.String()
	if countKey(line, LogKeyTraceID) != 1 || !strings.Contains(line, sc.TraceID().String()) {
		t.Errorf("want only the span's trace_id: %s", line)
	}
}
