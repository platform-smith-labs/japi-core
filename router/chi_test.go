package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/platform-smith-labs/japi-core/router"
)

// defaultMethods is the complete set that defaultRouterConfig must expose.
var defaultMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// nopHandler is a trivial HTTP handler used to register test routes so that
// chi's middleware chain (including CORS) runs for those paths.
var nopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// sendPreflight sends a CORS preflight OPTIONS request for the given method.
func sendPreflight(t *testing.T, r http.Handler, origin, requestMethod string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", requestMethod)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestDefaultConfig_AllowedMethodsIncludePatchAndHead verifies that
// defaultRouterConfig includes all 7 expected HTTP methods.
// Uses the test-only DefaultAllowedMethods helper to check config directly
// (go-chi/cors echoes back only the requested method in preflight responses,
// not the full list, so the full default list must be validated at config level).
func TestDefaultConfig_AllowedMethodsIncludePatchAndHead(t *testing.T) {
	_ = router.NewChiRouter() // must construct without panic
	methods := router.DefaultAllowedMethods()
	for _, m := range defaultMethods {
		found := false
		for _, am := range methods {
			if am == m {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("default AllowedMethods missing %q; got %v", m, methods)
		}
	}
}

// TestNewChiRouterWithCORS_AllowsPatchPreflight verifies that NewChiRouterWithCORS
// allows a PATCH preflight because PATCH is in the default AllowedMethods.
// go-chi/cors echoes back the requested method when the preflight is accepted.
func TestNewChiRouterWithCORS_AllowsPatchPreflight(t *testing.T) {
	origin := "https://app.example.com"
	r := router.NewChiRouterWithCORS([]string{origin})
	r.Patch("/", nopHandler) // register route so chi's middleware chain runs

	w := sendPreflight(t, r, origin, "PATCH")

	// go-chi/cors echoes back the requested method if allowed
	methods := w.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "PATCH") {
		t.Errorf("Access-Control-Allow-Methods = %q; want it to contain PATCH (PATCH must be in default allowed methods)", methods)
	}
}

// TestNewChiRouterWithOptions_RejectsUnconfiguredMethod verifies that
// WithAllowedMethods prevents methods not in the custom list from passing preflight.
func TestNewChiRouterWithOptions_RejectsUnconfiguredMethod(t *testing.T) {
	origin := "https://app.example.com"
	r := router.NewChiRouterWithOptions(
		router.WithAllowedOrigins([]string{origin}),
		router.WithAllowedMethods([]string{"GET", "POST"}),
	)
	r.Get("/", nopHandler)

	w := sendPreflight(t, r, origin, "PATCH")

	// PATCH is not in the custom list — preflight must be rejected (no CORS headers)
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("Access-Control-Allow-Methods = %q; want empty (PATCH not in custom list)", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty (PATCH not in custom list)", got)
	}
}

// TestNewChiRouterWithOptions_OriginAndMethods verifies that both
// WithAllowedOrigins and WithAllowedMethods are applied and reflected
// in the preflight response.
func TestNewChiRouterWithOptions_OriginAndMethods(t *testing.T) {
	origin := "https://a.com"
	r := router.NewChiRouterWithOptions(
		router.WithAllowedOrigins([]string{origin}),
		router.WithAllowedMethods([]string{"GET", "PATCH"}),
	)
	r.Patch("/", nopHandler)

	w := sendPreflight(t, r, origin, "PATCH")

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("Access-Control-Allow-Origin = %q; want %q", got, origin)
	}
	if methods := w.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "PATCH") {
		t.Errorf("Access-Control-Allow-Methods = %q; want it to contain PATCH", methods)
	}
}
