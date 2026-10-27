package typed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/handler"
)

const marker = "MARKER-91ab"

type item struct {
	SKU string `json:"sku" validate:"required,min=3"`
}

type createBody struct {
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
	Items []item `json:"items" validate:"dive"`
}

type listParams struct {
	Limit int `query:"limit" validate:"max=100"`
}

func problemOf(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != core.ProblemContentType {
		t.Fatalf("Content-Type = %q, body %s", ct, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func setBase(t *testing.T) {
	core.SetTypeBase("https://errors.example.test/")
	t.Cleanup(func() { core.SetTypeBase("") })
}

func TestParseBodyInvalidJSONDoesNotEchoInput(t *testing.T) {
	setBase(t)
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	h := handler.AdaptHandler(nil, logger, WithLogging(ParseBody(
		func(handler.HandlerContext[struct{}, createBody], http.ResponseWriter, *http.Request) (struct{}, error) {
			t.Fatal("handler reached")
			return struct{}{}, nil
		})))
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name": `+marker)))

	body := problemOf(t, rec)
	if rec.Code != 400 || body["type"] != "https://errors.example.test/invalid_json" {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("input echoed: %s", rec.Body.String())
	}
	// The typed access log line describes the error by code, not by text.
	for line := range strings.SplitSeq(strings.TrimSpace(logs.String()), "\n") {
		if strings.Contains(line, "HTTP Response Error") && strings.Contains(line, "invalid character") {
			t.Errorf("access log quotes the error text: %s", line)
		}
	}
	if !strings.Contains(logs.String(), `"error.type":"invalid_json"`) {
		t.Errorf("error.type missing from the log: %s", logs.String())
	}
}

func TestParseBodyValidationPointers(t *testing.T) {
	setBase(t)
	h := handler.AdaptHandler(nil, slog.New(slog.DiscardHandler), ParseBody(
		func(handler.HandlerContext[struct{}, createBody], http.ResponseWriter, *http.Request) (struct{}, error) {
			return struct{}{}, nil
		}))
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(`{"name":"ab","email":"`+marker+`","items":[{"sku":"abcd"},{"sku":"x"}]}`)))

	body := problemOf(t, rec)
	if rec.Code != 422 || body["type"] != "https://errors.example.test/validation_failed" {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("rejected value echoed: %s", rec.Body.String())
	}
	got := map[string]string{}
	for _, e := range body["errors"].([]any) {
		m := e.(map[string]any)
		got[m["pointer"].(string)] = m["detail"].(string)
	}
	want := map[string]string{
		"#/name":        "name must be at least 3 characters",
		"#/email":       "email must be a valid email address",
		"#/items/1/sku": "sku must be at least 3 characters",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("errors = %v, want %v", got, want)
	}
}

func TestParseParamsValidationHasNoPointer(t *testing.T) {
	setBase(t)
	h := handler.AdaptHandler(nil, slog.New(slog.DiscardHandler), ParseParams(
		func(handler.HandlerContext[listParams, struct{}], http.ResponseWriter, *http.Request) (struct{}, error) {
			return struct{}{}, nil
		}))

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/?limit=500", nil))
	body := problemOf(t, rec)
	errs := body["errors"].([]any)
	if rec.Code != 422 || len(errs) != 1 {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if _, has := errs[0].(map[string]any)["pointer"]; has {
		t.Errorf("query parameter error carries a pointer: %v", errs[0])
	}

	rec = httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/?limit="+marker, nil))
	if rec.Code != 400 || strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestAdapterUnknownErrorAndTimeout(t *testing.T) {
	setBase(t)
	var logs bytes.Buffer
	run := func(err error) *httptest.ResponseRecorder {
		h := handler.AdaptHandler(nil, slog.New(slog.NewJSONHandler(&logs, nil)),
			func(handler.HandlerContext[struct{}, struct{}], http.ResponseWriter, *http.Request) (struct{}, error) {
				return struct{}{}, err
			})
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec
	}

	rec := run(fmt.Errorf("pq: %s", marker))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if problemOf(t, rec)["type"] != "https://errors.example.test/internal_error" {
		t.Fatalf("body %s", rec.Body.String())
	}
	if strings.Count(logs.String(), marker) != 1 {
		t.Errorf("cause should be logged exactly once: %s", logs.String())
	}

	rec = run(fmt.Errorf("query: %w", context.DeadlineExceeded))
	if rec.Code != 504 || problemOf(t, rec)["type"] != "https://errors.example.test/timeout" {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}

	rec = run(fmt.Errorf("query: %w", context.Canceled))
	if rec.Body.Len() != 0 {
		t.Fatalf("canceled request got a body: %s", rec.Body.String())
	}

	rec = run(errors.Join(errors.New("x"), core.Conflict(core.CodeConflict, "Already exists.")))
	if rec.Code != 409 {
		t.Fatalf("joined problem not found: %d %s", rec.Code, rec.Body.String())
	}
}

func TestJSONPointer(t *testing.T) {
	for in, want := range map[string]string{
		"Body.name":              "#/name",
		"Body.items[2].sku":      "#/items/2/sku",
		"Body.matrix[1][0]":      "#/matrix/1/0",
		"Body.labels[a/b~c]":     "#/labels/a~1b~0c",
		"Body.address.post_code": "#/address/post_code",
		"NoDot":                  "",
	} {
		if got := jsonPointer(in); got != want {
			t.Errorf("jsonPointer(%q) = %q, want %q", in, got, want)
		}
	}
}

type csvRow struct {
	Name  string `csv:"name" validate:"required"`
	Email string `csv:"email" validate:"required,email"`
}

// upload builds a multipart request carrying content as the "file" field.
func upload(t *testing.T, filename, content string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte(content))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadErrorsHaveFixedDetails(t *testing.T) {
	setBase(t)
	csvH := handler.AdaptHandler(nil, slog.New(slog.DiscardHandler), ParseCSV(
		func(handler.HandlerContext[struct{}, []csvRow], http.ResponseWriter, *http.Request) (struct{}, error) {
			return struct{}{}, nil
		}))
	jsonH := handler.AdaptHandler(nil, slog.New(slog.DiscardHandler), ParseJSON(
		func(handler.HandlerContext[struct{}, createBody], http.ResponseWriter, *http.Request) (struct{}, error) {
			return struct{}{}, nil
		}))

	tooLarge := upload(t, "a.csv", strings.Repeat("x", 4096))
	tooLarge.Body = http.MaxBytesReader(httptest.NewRecorder(), tooLarge.Body, 64)

	notMultipart := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(marker))
	notMultipart.Header.Set("Content-Type", "text/plain")

	tests := []struct {
		name   string
		h      http.HandlerFunc
		req    *http.Request
		status int
		code   string
	}{
		{"csv wrong extension", csvH, upload(t, "a.txt", "name,email\n"), 415, "unsupported_media_type"},
		{"csv malformed", csvH, upload(t, "a.csv", "name,email\n\"unterminated "+marker+"\n"), 400, "invalid_csv"},
		{"csv empty", csvH, upload(t, "a.csv", "name,email\n"), 400, "invalid_csv"},
		{"csv row invalid", csvH, upload(t, "a.csv", "name,email\nbob,"+marker+"\n"), 422, "validation_failed"},
		{"too large", csvH, tooLarge, 413, "file_too_large"},
		{"not multipart", csvH, notMultipart, 415, "unsupported_media_type"},
		{"json malformed", jsonH, upload(t, "a.json", `{"name": `+marker), 400, "invalid_json"},
		{"json wrong extension", jsonH, upload(t, "a.csv", `{}`), 415, "unsupported_media_type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.h(rec, tt.req)
			body := problemOf(t, rec)
			if rec.Code != tt.status || body["type"] != "https://errors.example.test/"+tt.code {
				t.Fatalf("got %d %v", rec.Code, body)
			}
			if strings.Contains(rec.Body.String(), marker) {
				t.Fatalf("upload content echoed: %s", rec.Body.String())
			}
		})
	}
}

type ageRow struct {
	Name string `csv:"name"`
	Age  int    `csv:"age"`
}

// A client error's cause can quote customer content; only its type may be
// logged.
func TestClientErrorCauseTextNeverLogged(t *testing.T) {
	setBase(t)
	const sentinel = "SENTINEL-CELL-4e1d"
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	csvH := handler.AdaptHandler(nil, logger, WithLogging(ParseCSV(
		func(handler.HandlerContext[struct{}, []ageRow], http.ResponseWriter, *http.Request) (struct{}, error) {
			t.Fatal("handler reached")
			return struct{}{}, nil
		})))
	jsonH := handler.AdaptHandler(nil, logger, WithLogging(ParseBody(
		func(handler.HandlerContext[struct{}, createBody], http.ResponseWriter, *http.Request) (struct{}, error) {
			t.Fatal("handler reached")
			return struct{}{}, nil
		})))
	paramH := handler.AdaptHandler(nil, logger, WithLogging(ParseParams(
		func(handler.HandlerContext[listParams, struct{}], http.ResponseWriter, *http.Request) (struct{}, error) {
			t.Fatal("handler reached")
			return struct{}{}, nil
		})))

	for name, run := range map[string]func() *httptest.ResponseRecorder{
		"csv cell": func() *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			csvH(rec, upload(t, "a.csv", "name,age\nbob,"+sentinel+"\n"))
			return rec
		},
		"json fragment": func() *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			jsonH(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name": 5, "items": "`+sentinel+`"}`)))
			return rec
		},
		"param value": func() *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			paramH(rec, httptest.NewRequest(http.MethodGet, "/?limit="+sentinel, nil))
			return rec
		},
	} {
		t.Run(name, func(t *testing.T) {
			logs.Reset()
			rec := run()
			if rec.Code < 400 || rec.Code >= 500 {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), sentinel) {
				t.Errorf("sentinel in body: %s", rec.Body.String())
			}
			if strings.Contains(logs.String(), sentinel) {
				t.Errorf("sentinel in log output:\n%s", logs.String())
			}
			if !strings.Contains(logs.String(), `"error.cause_type"`) {
				t.Errorf("cause type not logged:\n%s", logs.String())
			}
		})
	}
}
