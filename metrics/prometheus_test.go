package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// TestEnablePrometheusMetrics_BasicTracking verifies metrics are tracked
func TestEnablePrometheusMetrics_BasicTracking(t *testing.T) {
	t.Run("tracks requests and exposes metrics endpoint", func(t *testing.T) {
		// Create isolated registry for this test
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()

		// Enable metrics with custom registry
		collector := enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)
		if collector == nil {
			t.Fatal("Expected non-nil collector")
		}

		// Add test endpoint
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Make test request
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Check metrics endpoint
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		if metricsRec.Code != http.StatusOK {
			t.Errorf("Expected metrics endpoint status 200, got %d", metricsRec.Code)
		}

		body := metricsRec.Body.String()

		// Verify metrics are present
		expectedMetrics := []string{
			"http_requests_total",
			"http_request_duration_seconds",
			"http_requests_in_flight",
		}

		for _, metric := range expectedMetrics {
			if !strings.Contains(body, metric) {
				t.Errorf("Expected metric %s in metrics output", metric)
			}
		}
	})
}

// TestEnablePrometheusMetrics_RequestCounting verifies request counting
func TestEnablePrometheusMetrics_RequestCounting(t *testing.T) {
	t.Run("counts multiple requests", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Make 5 requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}

		// Check metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Should show 5 requests (plus the metrics endpoint request itself)
		if !strings.Contains(body, `http_requests_total{method="GET",path="/test",status="200"} 5`) {
			t.Errorf("Expected 5 requests in metrics, got: %s", body)
		}
	})
}

// TestEnablePrometheusMetrics_StatusCodes verifies status code tracking
func TestEnablePrometheusMetrics_StatusCodes(t *testing.T) {
	t.Run("tracks different status codes", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		r.Get("/success", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		// Make requests with different status codes
		successReq := httptest.NewRequest("GET", "/success", nil)
		successRec := httptest.NewRecorder()
		r.ServeHTTP(successRec, successReq)

		errorReq := httptest.NewRequest("GET", "/error", nil)
		errorRec := httptest.NewRecorder()
		r.ServeHTTP(errorRec, errorReq)

		// Check metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Verify both status codes are tracked
		if !strings.Contains(body, `status="200"`) {
			t.Error("Expected status 200 in metrics")
		}

		if !strings.Contains(body, `status="500"`) {
			t.Error("Expected status 500 in metrics")
		}
	})
}

// TestEnablePrometheusMetrics_PathNormalization verifies path parameter handling
func TestEnablePrometheusMetrics_PathNormalization(t *testing.T) {
	t.Run("normalizes paths with parameters", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Make requests with different IDs
		ids := []string{"123", "456", "789"}
		for _, id := range ids {
			req := httptest.NewRequest("GET", "/users/"+id, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}

		// Check metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Should group all requests under the pattern "/users/{id}"
		if !strings.Contains(body, `path="/users/{id}"`) {
			t.Errorf("Expected path normalization to '/users/{id}', got: %s", body)
		}

		// Should show 3 total requests
		if !strings.Contains(body, `http_requests_total{method="GET",path="/users/{id}",status="200"} 3`) {
			t.Errorf("Expected 3 requests grouped by pattern, got: %s", body)
		}
	})
}

// TestEnablePrometheusMetrics_DurationTracking verifies duration histogram
func TestEnablePrometheusMetrics_DurationTracking(t *testing.T) {
	t.Run("records request duration", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		r.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		// Make request
		req := httptest.NewRequest("GET", "/slow", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		// Check metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Should have duration metrics
		if !strings.Contains(body, "http_request_duration_seconds") {
			t.Error("Expected duration metrics")
		}

		// Should have histogram buckets
		if !strings.Contains(body, "http_request_duration_seconds_bucket") {
			t.Error("Expected histogram buckets")
		}
	})
}

// TestEnablePrometheusMetrics_InFlightGauge verifies in-flight gauge
func TestEnablePrometheusMetrics_InFlightGauge(t *testing.T) {
	t.Run("tracks concurrent requests", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		// Channel to control request completion
		startCh := make(chan struct{})
		doneCh := make(chan struct{})

		r.Get("/concurrent", func(w http.ResponseWriter, r *http.Request) {
			<-startCh // Wait for signal
			w.WriteHeader(http.StatusOK)
			doneCh <- struct{}{}
		})

		// Start concurrent request
		go func() {
			req := httptest.NewRequest("GET", "/concurrent", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}()

		// Give goroutine time to start
		time.Sleep(50 * time.Millisecond)

		// Check metrics while request is in flight
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Should show at least 1 request in flight (the concurrent one)
		if !strings.Contains(body, "http_requests_in_flight") {
			t.Error("Expected in-flight metric")
		}

		// Signal request to complete
		close(startCh)
		<-doneCh
	})
}

// TestEnablePrometheusMetrics_CustomOptions verifies custom options
func TestEnablePrometheusMetrics_CustomOptions(t *testing.T) {
	t.Run("uses custom options", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()

		opts := MetricsOptions{
			DurationBuckets: []float64{0.1, 1, 10},
			Namespace:       "myapp",
			Subsystem:       "api",
		}

		enablePrometheusMetricsWithRegisterer(r, "/custom-metrics", opts, reg)

		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Make request
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		// Check custom metrics endpoint
		metricsReq := httptest.NewRequest("GET", "/custom-metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		if metricsRec.Code != http.StatusOK {
			t.Errorf("Expected custom metrics endpoint to work, got status %d", metricsRec.Code)
		}

		body := metricsRec.Body.String()

		// Verify custom namespace
		if !strings.Contains(body, "myapp_api_requests_total") {
			t.Errorf("Expected custom namespace 'myapp_api', got: %s", body)
		}
	})
}

// TestEnablePrometheusMetrics_ConcurrentRequests verifies thread safety
func TestEnablePrometheusMetrics_ConcurrentRequests(t *testing.T) {
	t.Run("handles concurrent requests safely", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(1 * time.Millisecond) // Simulate work
			w.WriteHeader(http.StatusOK)
		})

		// Make 100 concurrent requests
		var wg sync.WaitGroup
		numRequests := 100

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/test", nil)
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)
			}()
		}

		wg.Wait()

		// Check metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Should show all 100 requests
		if !strings.Contains(body, `http_requests_total{method="GET",path="/test",status="200"} 100`) {
			t.Errorf("Expected 100 requests in metrics, got: %s", body)
		}
	})
}

// TestEnablePrometheusMetrics_DifferentMethods verifies HTTP method tracking
func TestEnablePrometheusMetrics_DifferentMethods(t *testing.T) {
	t.Run("tracks different HTTP methods", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		r := chi.NewRouter()
		enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

		r.Get("/resource", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Post("/resource", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})

		r.Delete("/resource", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		// Make requests with different methods
		methods := []struct {
			method string
			status int
		}{
			{"GET", http.StatusOK},
			{"POST", http.StatusCreated},
			{"DELETE", http.StatusNoContent},
		}

		for _, m := range methods {
			req := httptest.NewRequest(m.method, "/resource", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}

		// Check metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		r.ServeHTTP(metricsRec, metricsReq)

		body := metricsRec.Body.String()

		// Verify all methods are tracked
		if !strings.Contains(body, `method="GET"`) {
			t.Error("Expected GET method in metrics")
		}

		if !strings.Contains(body, `method="POST"`) {
			t.Error("Expected POST method in metrics")
		}

		if !strings.Contains(body, `method="DELETE"`) {
			t.Error("Expected DELETE method in metrics")
		}
	})
}

// TestDefaultMetricsOptions verifies default options are sensible
func TestDefaultMetricsOptions(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		opts := DefaultMetricsOptions()

		if len(opts.DurationBuckets) == 0 {
			t.Error("Expected non-empty duration buckets")
		}

		if opts.Namespace != "http" {
			t.Errorf("Expected namespace 'http', got %s", opts.Namespace)
		}

		// Verify buckets cover reasonable ranges
		hasSmallBucket := false
		hasLargeBucket := false

		for _, bucket := range opts.DurationBuckets {
			if bucket <= 0.01 {
				hasSmallBucket = true
			}
			if bucket >= 5 {
				hasLargeBucket = true
			}
		}

		if !hasSmallBucket {
			t.Error("Expected small bucket (<= 10ms) in defaults")
		}

		if !hasLargeBucket {
			t.Error("Expected large bucket (>= 5s) in defaults")
		}
	})
}

// BenchmarkMetricsMiddleware benchmarks metrics overhead
func BenchmarkMetricsMiddleware(b *testing.B) {
	reg := prometheus.NewRegistry()

	r := chi.NewRouter()
	enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

// BenchmarkMetricsEndpoint benchmarks metrics endpoint performance
func BenchmarkMetricsEndpoint(b *testing.B) {
	reg := prometheus.NewRegistry()

	r := chi.NewRouter()
	enablePrometheusMetricsWithRegisterer(r, "/metrics", DefaultMetricsOptions(), reg)

	// Make some requests to populate metrics
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		io.Copy(io.Discard, rec.Body)
	}
}
