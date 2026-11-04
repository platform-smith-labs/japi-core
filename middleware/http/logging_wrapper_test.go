package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_ResponseController(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}
	rc := http.NewResponseController(rw)
	if err := rc.Flush(); err != nil {
		t.Errorf("Flush: %v", err)
	}
	if !rec.Flushed {
		t.Error("underlying recorder was not flushed")
	}
	if _, _, err := rc.Hijack(); !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("Hijack on a non-hijackable writer: got %v, want ErrNotSupported", err)
	}
	if rw.Unwrap() != rec {
		t.Error("Unwrap does not return the wrapped writer")
	}
}

type failingFlushWriter struct{ *httptest.ResponseRecorder }

func (w failingFlushWriter) FlushError() error { return errFlushFailed }

var errFlushFailed = errors.New("flush failed")

func TestResponseWriter_FlushError(t *testing.T) {
	rw := &responseWriter{ResponseWriter: failingFlushWriter{httptest.NewRecorder()}, statusCode: http.StatusOK}
	if err := http.NewResponseController(rw).Flush(); !errors.Is(err, errFlushFailed) {
		t.Errorf("Flush() = %v, want %v", err, errFlushFailed)
	}
}
