package router_test

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

// TestRouterAndOtelLinkNoSDK guards the package split: router and otel may
// depend on the OpenTelemetry API and otelhttp only. The SDK, exporters and
// the Prometheus client belong to otel/sdk (and metrics), which a service
// imports explicitly.
func TestRouterAndOtelLinkNoSDK(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not available")
	}
	cmd := exec.Command(goBin, "list", "-deps", "./router", "./otel")
	cmd.Dir = ".." // module root
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go list failed: %v", err)
	}
	forbidden := []string{
		"go.opentelemetry.io/otel/sdk",
		"go.opentelemetry.io/otel/exporters",
		"github.com/prometheus/client_golang",
	}
	for _, pkg := range strings.Fields(string(out)) {
		for _, f := range forbidden {
			if pkg == f || strings.HasPrefix(pkg, f+"/") {
				t.Errorf("router/otel depend on %s", pkg)
			}
		}
	}
}

// TestNoPprofOnDefaultServeMux guards against importing net/http/pprof (for
// example through chi's middleware package), which registers profiling
// handlers on http.DefaultServeMux in every consumer. Profiling belongs on an
// explicit admin listener the service chooses.
func TestNoPprofOnDefaultServeMux(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	if _, pattern := http.DefaultServeMux.Handler(req); pattern != "" {
		t.Fatalf("http.DefaultServeMux serves %q: a package imports net/http/pprof", pattern)
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not available")
	}
	cmd := exec.Command(goBin, "list", "-deps", "./...")
	cmd.Dir = ".." // module root
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go list failed: %v", err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == "net/http/pprof" || pkg == "github.com/go-chi/chi/v5/middleware" {
			t.Errorf("a japi-core package depends on %s", pkg)
		}
	}
}
