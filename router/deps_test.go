package router_test

import (
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
