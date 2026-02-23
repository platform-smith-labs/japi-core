package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector holds Prometheus metrics collectors for HTTP requests
type Collector struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
	registry         prometheus.Registerer
}

// MetricsOptions configures Prometheus metrics collection
type MetricsOptions struct {
	// DurationBuckets defines histogram buckets for request duration (in seconds)
	// Default: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10]
	DurationBuckets []float64

	// Namespace is the Prometheus namespace for metrics
	// Default: "http"
	Namespace string

	// Subsystem is the Prometheus subsystem for metrics
	// Default: "" (empty)
	Subsystem string
}

// DefaultMetricsOptions returns sensible defaults for most applications
func DefaultMetricsOptions() MetricsOptions {
	return MetricsOptions{
		DurationBuckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
		Namespace:       "http",
		Subsystem:       "",
	}
}

// EnablePrometheusMetrics enables Prometheus metrics collection and exposes metrics endpoint.
//
// This function:
//   - Registers Prometheus middleware to track all HTTP requests
//   - Exposes metrics at the specified endpoint (e.g., "/metrics")
//   - Uses default options (see DefaultMetricsOptions)
//
// Metrics tracked:
//   - http_requests_total{method,path,status} - Total number of HTTP requests
//   - http_request_duration_seconds{method,path} - HTTP request latency distribution
//   - http_requests_in_flight - Current number of HTTP requests being served
//
// Example:
//
//	r := router.NewChiRouter()
//	metrics.EnablePrometheusMetrics(r, "/metrics")
//
// The metrics endpoint will be available at the specified path.
// Access it via: curl http://localhost:8080/metrics
//
// Returns the Collector for advanced usage (e.g., custom metrics)
func EnablePrometheusMetrics(router chi.Router, metricsPath string) *Collector {
	return EnablePrometheusMetricsWithOptions(router, metricsPath, DefaultMetricsOptions())
}

// EnablePrometheusMetricsWithOptions enables Prometheus metrics with custom options.
//
// This allows fine-tuning of histogram buckets, namespace, and subsystem.
//
// Example:
//
//	opts := metrics.MetricsOptions{
//	    DurationBuckets: []float64{0.001, 0.01, 0.1, 1, 10},
//	    Namespace:       "myapp",
//	    Subsystem:       "api",
//	}
//	metrics.EnablePrometheusMetricsWithOptions(r, "/metrics", opts)
//
// Returns the Collector for advanced usage
func EnablePrometheusMetricsWithOptions(router chi.Router, metricsPath string, opts MetricsOptions) *Collector {
	return enablePrometheusMetricsWithRegisterer(router, metricsPath, opts, prometheus.DefaultRegisterer)
}

// enablePrometheusMetricsWithRegisterer allows injection of a custom registerer for testing
func enablePrometheusMetricsWithRegisterer(
	router chi.Router,
	metricsPath string,
	opts MetricsOptions,
	registerer prometheus.Registerer,
) *Collector {
	// Create metrics collector
	collector := &Collector{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: opts.Namespace,
				Subsystem: opts.Subsystem,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: opts.Namespace,
				Subsystem: opts.Subsystem,
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency distribution",
				Buckets:   opts.DurationBuckets,
			},
			[]string{"method", "path"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: opts.Namespace,
				Subsystem: opts.Subsystem,
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being served",
			},
		),
		registry: registerer,
	}

	// Register metrics with Prometheus
	registerer.MustRegister(collector.requestsTotal)
	registerer.MustRegister(collector.requestDuration)
	registerer.MustRegister(collector.requestsInFlight)

	// Apply metrics middleware to router
	router.Use(collector.middleware)

	// Expose metrics endpoint using custom registry if not default
	if reg, ok := registerer.(*prometheus.Registry); ok && reg != prometheus.DefaultRegisterer {
		// Use custom gatherer for test registries
		handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		router.Handle(metricsPath, handler)
	} else {
		// Use default handler for production
		router.Handle(metricsPath, promhttp.Handler())
	}

	return collector
}

// middleware tracks HTTP request metrics
func (c *Collector) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment in-flight requests
		c.requestsInFlight.Inc()
		defer c.requestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(ww, r)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get route pattern (normalized path with placeholders)
		routePattern := getRoutePattern(r)

		// Record metrics
		c.requestsTotal.WithLabelValues(
			r.Method,
			routePattern,
			strconv.Itoa(ww.statusCode),
		).Inc()

		c.requestDuration.WithLabelValues(
			r.Method,
			routePattern,
		).Observe(duration)
	})
}

// getRoutePattern extracts the route pattern from chi's route context
// This normalizes paths like "/users/123" to "/users/{id}" to prevent metric cardinality explosion
func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	// Fallback to actual path if route pattern not available
	return r.URL.Path
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write ensures status code is captured even if WriteHeader isn't explicitly called
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
