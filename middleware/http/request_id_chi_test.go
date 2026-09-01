package http_test

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	japihttp "github.com/platform-smith-labs/japi-core/v3/middleware/http"
)

// The regression this package existed to have: chi's middleware.RequestID is what the
// default router installs, and GetRequestID used to read a different key — so it
// returned "" on every request while an ID sat in the context, unused.
func TestGetRequestID_ReadsChiKey(t *testing.T) {
	var got string
	h := chimiddleware.RequestID(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		got = japihttp.GetRequestID(r)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(nethttp.MethodGet, "/", nil))

	if got == "" {
		t.Fatal("GetRequestID returned empty for a request carrying chi's request ID")
	}
}

// An inbound X-Request-ID must be honoured, not replaced — that is what lets one ID
// span a hop between services.
func TestGetRequestID_HonoursInboundHeader(t *testing.T) {
	const want = "caller-supplied-id"
	var got string
	h := chimiddleware.RequestID(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		got = japihttp.GetRequestID(r)
	}))
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	req.Header.Set(japihttp.RequestIDHeader, want)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got != want {
		t.Fatalf("inbound request ID not propagated: got %q, want %q", got, want)
	}
}

// This package's own middleware must still win when both are installed: it is the more
// specific choice, having been added deliberately.
func TestGetRequestID_OwnMiddlewareTakesPrecedence(t *testing.T) {
	const want = "explicit-id"
	var got string
	inner := nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		got = japihttp.GetRequestID(r)
	})
	h := chimiddleware.RequestID(japihttp.WithRequestID()(inner))
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	req.Header.Set(japihttp.RequestIDHeader, want)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// BACKWARD COMPATIBILITY. RequestIDContextKey is a published untyped string constant.
// A caller may hold it in a string, and a caller may read ctx.Value("request_id")
// directly. Changing its type would break both silently — this test is what stops that
// happening in a patch release.
func TestRequestIDContextKey_RemainsAPlainString(t *testing.T) {
	var key string = japihttp.RequestIDContextKey
	if key != "request_id" {
		t.Fatalf("published context key changed: %q", key)
	}

	ctx := context.WithValue(context.Background(), japihttp.RequestIDContextKey, "abc") //nolint:staticcheck // pinning published behaviour
	if got, _ := ctx.Value("request_id").(string); got != "abc" {
		t.Fatal("a bare-string read of the context key no longer works — this is a breaking change")
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

// The response header is the half chi does not do, and the half a human needs.
func TestWithRequestIDHeader_EchoesOntoResponse(t *testing.T) {
	h := chimiddleware.RequestID(japihttp.WithRequestIDHeader()(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			w.WriteHeader(nethttp.StatusTeapot)
		})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/", nil))

	if rec.Header().Get(japihttp.RequestIDHeader) == "" {
		t.Fatal("X-Request-ID was not echoed onto the response")
	}
	if rec.Code != nethttp.StatusTeapot {
		t.Fatalf("handler status not preserved: %d", rec.Code)
	}
}

// A handler that sets the header deliberately must win.
func TestWithRequestIDHeader_DoesNotOverwriteHandler(t *testing.T) {
	const want = "handler-chose-this"
	h := chimiddleware.RequestID(japihttp.WithRequestIDHeader()(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			w.Header().Set(japihttp.RequestIDHeader, want)
		})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/", nil))

	if got := rec.Header().Get(japihttp.RequestIDHeader); got != want {
		t.Fatalf("middleware overwrote the handler's header: got %q", got)
	}
}

// The default router must wire both halves — this is what gives every consumer the
// behaviour on a version bump alone, with no source change.
func TestChiRouterEchoesRequestIDEndToEnd(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(japihttp.WithRequestIDHeader())
	r.Get("/x", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if japihttp.GetRequestID(r) == "" {
			t.Error("handler saw no request ID")
		}
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/x", nil))

	if rec.Header().Get(japihttp.RequestIDHeader) == "" {
		t.Fatal("router did not echo X-Request-ID")
	}
}
