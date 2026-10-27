package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/platform-smith-labs/japi-core/v4/core"
)

const (
	testBase = "https://errors.example.test/"
	marker   = "MARKER-7f3c-secret"
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// testRegistry is a small registry with one code declaring an extension.
var testRegistry = core.RegistryFunc(func(c core.Code) (core.CodeInfo, bool) {
	switch c {
	case "widget_missing":
		return core.CodeInfo{Title: "Widget not found", Status: 404}, true
	case "version_conflict":
		return core.CodeInfo{Title: "Version conflict", Status: 409, Extensions: []string{"current_version"}}, true
	}
	return core.CodeInfo{}, false
})

// harness serves h behind a recording span and captures the log.
type harness struct {
	spans *tracetest.SpanRecorder
	tp    *sdktrace.TracerProvider
	logs  *bytes.Buffer
	log   *slog.Logger
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	spans := tracetest.NewSpanRecorder()
	buf := &bytes.Buffer{}
	return &harness{
		spans: spans,
		tp:    sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)),
		logs:  buf,
		log:   slog.New(slog.NewJSONHandler(buf, nil)),
	}
}

// serve runs write inside a span and returns the recorder and the span's trace ID.
func (h *harness) serve(t *testing.T, write func(w http.ResponseWriter, r *http.Request)) (*httptest.ResponseRecorder, string) {
	t.Helper()
	ctx, span := h.tp.Tracer("test").Start(context.Background(), "request")
	req := httptest.NewRequest(http.MethodPost, "/widgets/42?q=1", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	write(rec, req)
	span.End()
	return rec, span.SpanContext().TraceID().String()
}

func withRegistry(t *testing.T, r core.Registry, base string) {
	t.Helper()
	core.SetCodeRegistry(r)
	core.SetTypeBase(base)
	t.Cleanup(func() {
		core.SetCodeRegistry(nil)
		core.SetTypeBase("")
	})
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != core.ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, core.ProblemContentType)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, rec.Body.String())
	}
	checkRFC9457(t, body)
	return body
}

// checkRFC9457 checks the member types RFC 9457 section 3 defines, and this
// package's standard extensions.
func checkRFC9457(t *testing.T, body map[string]any) {
	t.Helper()
	str := func(k string, required bool) {
		v, ok := body[k]
		if !ok {
			if required {
				t.Errorf("member %q missing", k)
			}
			return
		}
		if _, isStr := v.(string); !isStr {
			t.Errorf("member %q is %T, want string", k, v)
		}
	}
	str("type", true)
	str("title", true)
	str("detail", false)
	str("instance", false)
	if s, ok := body["status"].(float64); !ok || s != float64(int(s)) || s < 400 || s > 599 {
		t.Errorf("status = %v, want an integer 400..599", body["status"])
	}
	if v, ok := body["trace_id"]; ok {
		if s, _ := v.(string); !traceIDPattern.MatchString(s) {
			t.Errorf("trace_id = %v, want 32 lowercase hex", v)
		}
	}
	if v, ok := body["errors"]; ok {
		list, isList := v.([]any)
		if !isList || len(list) == 0 {
			t.Fatalf("errors = %v, want a non-empty array", v)
		}
		for _, e := range list {
			m, isObj := e.(map[string]any)
			if !isObj {
				t.Fatalf("errors entry %v is not an object", e)
			}
			if _, ok := m["detail"].(string); !ok {
				t.Errorf("errors entry %v lacks a string detail", m)
			}
			if p, ok := m["pointer"]; ok {
				if s, _ := p.(string); !strings.HasPrefix(s, "#/") {
					t.Errorf("pointer %v is not a JSON Pointer fragment", p)
				}
			}
		}
	}
}

func TestGoldenBodies(t *testing.T) {
	withRegistry(t, testRegistry, testBase)

	tests := []struct {
		name    string
		problem *core.Problem
		want    string // with trace_id replaced by TRACE
	}{
		{
			name: "400 with errors",
			problem: core.BadRequest(core.CodeBadRequest, "Two fields are invalid.").WithErrors(
				core.FieldError{Detail: "name must be at least 3 characters", Pointer: "#/name"},
				core.FieldError{Detail: "limit must be at most 100"},
			),
			want: `{"detail":"Two fields are invalid.","errors":[{"detail":"name must be at least 3 characters","pointer":"#/name"},{"detail":"limit must be at most 100"}],"status":400,"title":"Bad request","trace_id":"TRACE","type":"https://errors.example.test/bad_request"}`,
		},
		{
			name:    "404 registered code",
			problem: core.NotFound("widget_missing", "No widget with that ID."),
			want:    `{"detail":"No widget with that ID.","status":404,"title":"Widget not found","trace_id":"TRACE","type":"https://errors.example.test/widget_missing"}`,
		},
		{
			name:    "409 with declared extension",
			problem: core.Conflict("version_conflict", "The widget changed.").WithExtension("current_version", 7),
			want:    `{"current_version":7,"detail":"The widget changed.","status":409,"title":"Version conflict","trace_id":"TRACE","type":"https://errors.example.test/version_conflict"}`,
		},
		{
			name:    "422 validation",
			problem: core.Validation(core.FieldError{Detail: "email must be a valid email address", Pointer: "#/email"}),
			want:    `{"errors":[{"detail":"email must be a valid email address","pointer":"#/email"}],"status":422,"title":"Validation failed","trace_id":"TRACE","type":"https://errors.example.test/validation_failed"}`,
		},
		{
			name:    "429",
			problem: core.TooManyRequests(core.CodeRateLimited, "Slow down."),
			want:    `{"detail":"Slow down.","status":429,"title":"Too many requests","trace_id":"TRACE","type":"https://errors.example.test/rate_limited"}`,
		},
		{
			name: "500 with cause and detail",
			problem: core.Internal(fmt.Errorf("pq: %s", marker)).WithDetail("db said " + marker).WithErrors(
				core.FieldError{Detail: marker}),
			want: `{"status":500,"title":"Internal server error","trace_id":"TRACE","type":"https://errors.example.test/internal_error"}`,
		},
		{
			name:    "503 constructed detail without cause",
			problem: core.NewProblem(core.CodeUnavailable, 503, "backend "+marker+" is down"),
			want:    `{"status":503,"title":"Service unavailable","trace_id":"TRACE","type":"https://errors.example.test/unavailable"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			rec, traceID := h.serve(t, func(w http.ResponseWriter, r *http.Request) {
				_ = core.WriteProblem(w, r, tt.problem, core.WithLogger(h.log))
			})
			body := decode(t, rec)
			if body["trace_id"] != traceID {
				t.Errorf("trace_id = %v, want the span's %s", body["trace_id"], traceID)
			}
			got := strings.TrimSpace(strings.Replace(rec.Body.String(), traceID, "TRACE", 1))
			if got != tt.want {
				t.Errorf("body\n got %s\nwant %s", got, tt.want)
			}
			if int(body["status"].(float64)) != rec.Code {
				t.Errorf("status member %v != response code %d", body["status"], rec.Code)
			}
		})
	}
}

func TestServerErrorCauseGoesToLogAndSpanOnly(t *testing.T) {
	withRegistry(t, nil, testBase)
	h := newHarness(t)
	rec, traceID := h.serve(t, func(w http.ResponseWriter, r *http.Request) {
		_ = core.WriteProblem(w, r, core.Internal(errors.New("pq: "+marker)).WithDetail("detail "+marker),
			core.WithLogger(h.log))
	})
	if strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("marker reached the body: %s", rec.Body.String())
	}
	logged := h.logs.String()
	if strings.Count(logged, "\n") != 1 {
		t.Errorf("want exactly one log line, got %q", logged)
	}
	if !strings.Contains(logged, "pq: "+marker) || !strings.Contains(logged, "detail "+marker) {
		t.Errorf("cause and detail missing from the log line: %s", logged)
	}
	if !strings.Contains(logged, `"level":"ERROR"`) {
		t.Errorf("5xx not logged at ERROR: %s", logged)
	}
	spans := h.spans.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d spans", len(spans))
	}
	s := spans[0]
	if s.SpanContext().TraceID().String() != traceID {
		t.Fatal("wrong span")
	}
	if s.Status().Code != codes.Error || s.Status().Description != "internal_error" {
		t.Errorf("span status = %v %q", s.Status().Code, s.Status().Description)
	}
	found := false
	for _, ev := range s.Events() {
		if ev.Name != "exception" {
			continue
		}
		for _, a := range ev.Attributes {
			if a.Key == "exception.message" && strings.Contains(a.Value.AsString(), marker) {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("cause not recorded as a span exception event: %+v", s.Events())
	}
}

func TestHandlerFuncUnknownErrorIsGeneric500(t *testing.T) {
	withRegistry(t, nil, testBase)
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	h := core.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		return fmt.Errorf("pq: %s", marker)
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != 500 {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), marker) || strings.Contains(rec.Body.String(), "pq") {
		t.Fatalf("error text leaked: %s", rec.Body.String())
	}
	body := decode(t, rec)
	if body["type"] != testBase+"internal_error" {
		t.Errorf("type = %v", body["type"])
	}
	if !strings.Contains(logs.String(), marker) {
		t.Errorf("cause not logged: %s", logs.String())
	}
}

func TestWrappedProblemIsFoundWithErrorsAs(t *testing.T) {
	withRegistry(t, nil, testBase)
	h := core.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		return fmt.Errorf("loading widget: %w", core.NotFound(core.CodeNotFound, "No such widget."))
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	body := decode(t, rec)
	if rec.Code != 404 || body["detail"] != "No such widget." {
		t.Fatalf("got %d %v", rec.Code, body)
	}
}

func TestContextCanceledWritesNothing(t *testing.T) {
	h := core.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		return fmt.Errorf("query: %w", context.Canceled)
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Body.Len() != 0 {
		t.Fatalf("wrote %q", rec.Body.String())
	}
}

func expectViolation(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		v := recover()
		if _, ok := v.(core.ContractViolation); !ok {
			t.Fatalf("want a ContractViolation panic, got %v", v)
		}
	}()
	f()
}

func TestContractViolationsPanicUnderTest(t *testing.T) {
	withRegistry(t, testRegistry, testBase)
	write := func(p *core.Problem) func() {
		return func() {
			rec := httptest.NewRecorder()
			_ = core.WriteProblem(rec, httptest.NewRequest(http.MethodGet, "/x", nil), p,
				core.WithLogger(slog.New(slog.DiscardHandler)))
		}
	}
	t.Run("unknown code", func(t *testing.T) { expectViolation(t, write(core.NotFound("no_such_code", ""))) })
	t.Run("status mismatch", func(t *testing.T) { expectViolation(t, write(core.BadRequest("widget_missing", ""))) })
	t.Run("undeclared extension", func(t *testing.T) {
		expectViolation(t, write(core.NotFound("widget_missing", "").WithExtension("state", "x")))
	})
	t.Run("reserved extension", func(t *testing.T) {
		expectViolation(t, write(core.Conflict("version_conflict", "").WithExtension("trace_id", "x")))
	})
	t.Run("invalid code", func(t *testing.T) { expectViolation(t, write(core.BadRequest("Bad Code", ""))) })
	t.Run("non-error status", func(t *testing.T) { expectViolation(t, write(core.NewProblem("widget_missing", 200, ""))) })
}

func TestBuiltinCodesResolveWithARegistry(t *testing.T) {
	withRegistry(t, testRegistry, testBase)
	for code, info := range core.BuiltinCodes() {
		rec := httptest.NewRecorder()
		_ = core.WriteProblem(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
			core.NewProblem(code, info.Status, ""), core.WithLogger(slog.New(slog.DiscardHandler)))
		if rec.Code != info.Status {
			t.Errorf("%s: status %d, want %d", code, rec.Code, info.Status)
		}
		body := decode(t, rec)
		if body["title"] != info.Title || body["type"] != testBase+string(code) {
			t.Errorf("%s: %v", code, body)
		}
	}
}

func TestAboutBlankWithoutTypeBase(t *testing.T) {
	withRegistry(t, nil, "")
	rec := httptest.NewRecorder()
	_ = core.WriteProblem(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
		core.NotFound("anything_goes", "nope"), core.WithLogger(slog.New(slog.DiscardHandler)))
	body := decode(t, rec)
	want := map[string]any{"type": "about:blank", "title": "Not Found", "status": float64(404), "detail": "nope"}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("got %v, want %v", body, want)
	}
}

func TestLegacyAPIErrorServerDetailSuppressed(t *testing.T) {
	var logs bytes.Buffer
	rec := httptest.NewRecorder()
	_ = core.WriteError(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
		core.NewAPIError(500, "boom "+marker, "pq: "+marker),
		core.WithLogger(slog.New(slog.NewJSONHandler(&logs, nil))))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logs.String(), marker) {
		t.Errorf("legacy 5xx text not logged: %s", logs.String())
	}

	rec = httptest.NewRecorder()
	_ = core.WriteError(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
		fmt.Errorf("wrapped: %w", core.NewAPIError(404, "Widget not found")),
		core.WithLogger(slog.New(slog.DiscardHandler)))
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"message":"Widget not found"`) {
		t.Fatalf("legacy 4xx envelope changed: %d %s", rec.Code, rec.Body.String())
	}
}
