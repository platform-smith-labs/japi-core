package http_test

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	japihttp "github.com/platform-smith-labs/japi-core/v4/middleware/http"
	"github.com/platform-smith-labs/japi-core/v4/router"
)

// withID stores id as the request ID, as an earlier middleware would.
func withID(id string, next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		next.ServeHTTP(w, r.WithContext(japihttp.ContextWithRequestID(r.Context(), id)))
	})
}

// An inbound X-Request-ID must be honoured, not replaced — that is what lets one ID
// span a hop between services.
func TestGetRequestID_HonoursInboundHeader(t *testing.T) {
	const want = "caller-supplied-id"
	var got string
	h := japihttp.WithRequestID()(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		got = japihttp.GetRequestID(r)
	}))
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	req.Header.Set(japihttp.RequestIDHeader, want)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got != want {
		t.Fatalf("inbound request ID not propagated: got %q, want %q", got, want)
	}
}

// RequestIDContextKey is a typed key: a bare string key set by another
// package cannot collide with it, and ContextWithRequestID round-trips.
func TestRequestIDContextKey_IsTyped(t *testing.T) {
	ctx := context.WithValue(context.Background(), "request_id", "bare") //nolint:staticcheck // the collision this guards against
	if got := japihttp.GetRequestIDFromContext(ctx); got == "bare" {
		t.Fatal("a bare string key is read as the request ID")
	}
	ctx = japihttp.ContextWithRequestID(ctx, "abc")
	if got := japihttp.GetRequestIDFromContext(ctx); got != "abc" {
		t.Fatalf("got %q, want abc", got)
	}
}

// No request ID anywhere is not an error; it is empty.
func TestGetRequestID_EmptyWhenAbsent(t *testing.T) {
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	if got := japihttp.GetRequestID(req); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := japihttp.GetRequestIDFromContext(nil); got != "" { //nolint:staticcheck // nil is a real caller mistake worth surviving
		t.Fatalf("nil context should yield empty, got %q", got)
	}
}

func TestWithRequestIDHeader_EchoesOntoResponse(t *testing.T) {
	h := withID("rid-1", japihttp.WithRequestIDHeader()(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			w.WriteHeader(nethttp.StatusTeapot)
		})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/", nil))

	if rec.Header().Get(japihttp.RequestIDHeader) != "rid-1" {
		t.Fatal("X-Request-ID was not echoed onto the response")
	}
	if rec.Code != nethttp.StatusTeapot {
		t.Fatalf("handler status not preserved: %d", rec.Code)
	}
}

// A handler that sets the header deliberately must win.
func TestWithRequestIDHeader_DoesNotOverwriteHandler(t *testing.T) {
	const want = "handler-chose-this"
	h := withID("rid-2", japihttp.WithRequestIDHeader()(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			w.Header().Set(japihttp.RequestIDHeader, want)
		})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/", nil))

	if got := rec.Header().Get(japihttp.RequestIDHeader); got != want {
		t.Fatalf("middleware overwrote the handler's header: got %q", got)
	}
}

// The default router (without tracing) gives every request an ID, visible to
// handlers and echoed on the response.
func TestChiRouterEchoesRequestIDEndToEnd(t *testing.T) {
	r := router.NewChiRouter()
	var seen string
	r.Get("/x", func(_ nethttp.ResponseWriter, r *nethttp.Request) { seen = japihttp.GetRequestID(r) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/x", nil))

	if seen == "" || rec.Header().Get(japihttp.RequestIDHeader) != seen {
		t.Fatalf("handler saw %q, header %q", seen, rec.Header().Get(japihttp.RequestIDHeader))
	}
}
