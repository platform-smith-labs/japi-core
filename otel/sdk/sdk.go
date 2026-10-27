// Package otelsdk wires the OpenTelemetry SDK into a service with one call.
//
// It lives apart from package otel, which depends only on the OpenTelemetry
// API, so that libraries and routers can be instrumented without linking the
// SDK, the Prometheus exporter or their transitive dependencies. Only a
// service's main package needs to import otelsdk.
//
// Importing it does nothing: no goroutines are started and nothing is
// registered in init(). Until Setup is called, the OpenTelemetry globals stay
// the API's no-op implementations, so instrumented code costs close to
// nothing.
//
// Setup installs, as process-wide globals:
//   - a W3C Trace Context propagator (traceparent/tracestate only, no baggage)
//   - a tracer provider that samples every span and exports through a bounded,
//     non-blocking batch processor (or exports nothing when no exporter is set)
//   - optionally, a meter provider whose instruments are exposed through a
//     caller-supplied Prometheus registerer
//
// Exporters are not constructed here. The OTLP exporter lives in the otel/otlp
// package so that binaries which never export do not link it. Inject it via
// Config.SpanExporter:
//
//	exp, err := otlp.FromEnv(ctx) // nil, nil when no OTLP endpoint is configured
//	if err != nil {
//	    return err
//	}
//	shutdown, err := otelsdk.Setup(ctx, otelsdk.Config{
//	    ServiceName:  "my-service",
//	    SpanExporter: exp,
//	    Registerer:   prometheus.DefaultRegisterer,
//	})
//	if err != nil {
//	    return err
//	}
//	defer shutdown(context.Background())
package otelsdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config configures Setup. The zero value is valid: it installs the propagator
// and a tracer provider that creates real trace IDs but exports nothing, and no
// meter provider.
type Config struct {
	// ServiceName is used as service.name only when neither OTEL_SERVICE_NAME
	// nor a service.name entry in OTEL_RESOURCE_ATTRIBUTES is set. The
	// environment always wins so operators can rename a deployment without a
	// rebuild.
	ServiceName string

	// Attributes are extra resource attributes, such as service.version or
	// deployment.environment.name. Like ServiceName they are fallbacks: a
	// key also set in OTEL_RESOURCE_ATTRIBUTES takes the environment's value.
	Attributes []attribute.KeyValue

	// SpanExporter receives finished spans through a batch span processor.
	// When nil, spans are still created, sampled and carry valid IDs (so they
	// can be propagated and correlated with logs), but nothing is exported.
	// See the otel/otlp package for an environment-configured exporter.
	SpanExporter sdktrace.SpanExporter

	// Registerer, when non-nil, enables the meter provider: metrics recorded
	// through the global MeterProvider are exposed by registering an
	// OpenTelemetry Prometheus exporter (a prometheus.Collector) here.
	// Temporality is cumulative, histograms use explicit buckets and
	// exemplars are attached for measurements taken inside a sampled span.
	// When nil, no meter provider is installed and the global stays no-op.
	Registerer prometheus.Registerer

	// MaxQueueSize bounds the number of finished spans buffered for export.
	// When the queue is full new spans are dropped; a request is never
	// blocked waiting on the exporter.
	// Default: SDK default (2048, or OTEL_BSP_MAX_QUEUE_SIZE).
	MaxQueueSize int

	// MaxExportBatchSize is the maximum number of spans sent per export call.
	// Default: SDK default (512, or OTEL_BSP_MAX_EXPORT_BATCH_SIZE).
	MaxExportBatchSize int

	// BatchTimeout is the longest a span waits in the queue before a batch is
	// sent, even if the batch is not full.
	// Default: SDK default (5s, or OTEL_BSP_SCHEDULE_DELAY).
	BatchTimeout time.Duration

	// ExportTimeout bounds a single export call.
	// Default: SDK default (30s, or OTEL_BSP_EXPORT_TIMEOUT).
	ExportTimeout time.Duration
}

// Setup builds the tracer provider (and, if Config.Registerer is set, the
// meter provider), installs them and the Trace Context propagator as the
// OpenTelemetry globals, and returns a shutdown function.
//
// shutdown flushes buffered spans and shuts down both providers, returning
// the joined errors. Call it once during graceful shutdown with a context
// that carries a deadline. Globals are only replaced when Setup succeeds.
//
// Setup is meant to be called once per process. Calling it again with the
// same Registerer fails, because the Prometheus collector is already
// registered.
func Setup(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	res, err := newResource(ctx, cfg.ServiceName, cfg.Attributes)
	switch {
	case errors.Is(err, resource.ErrPartialResource) && res != nil:
		// A malformed OTEL_RESOURCE_ATTRIBUTES must not stop the process:
		// keep what was detected and report the rest to the error handler.
		otelapi.Handle(err)
	case err != nil:
		return nil, fmt.Errorf("otelsdk: build resource: %w", err)
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		// Sample everything. A head-based ratio here would discard spans before
		// anyone knows whether the request was interesting; volume is better
		// controlled downstream (e.g. tail sampling in a collector). The
		// incoming sampled flag is deliberately not honoured, so a caller
		// cannot switch tracing off for this service.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}
	if cfg.SpanExporter != nil {
		tpOpts = append(tpOpts, sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(cfg.SpanExporter, batchOptions(cfg)...),
		))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)

	var mp *sdkmetric.MeterProvider
	if cfg.Registerer != nil {
		exp, err := otelprom.New(otelprom.WithRegisterer(cfg.Registerer))
		if err != nil {
			// The tracer provider has no running goroutines without an
			// exporter, but shut it down anyway to release the processor.
			_ = tp.Shutdown(ctx)
			return nil, fmt.Errorf("otelsdk: create prometheus exporter: %w", err)
		}
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(exp),
			// Attach exemplars only for measurements taken inside a sampled
			// span, so every exemplar links to a trace that was recorded.
			sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter),
		)
	}

	otelapi.SetTextMapPropagator(propagation.TraceContext{})
	otelapi.SetTracerProvider(tp)
	if mp != nil {
		otelapi.SetMeterProvider(mp)
	}

	return func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("otelsdk: shutdown tracer provider: %w", err))
		}
		if mp != nil {
			if err := mp.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("otelsdk: shutdown meter provider: %w", err))
			}
		}
		return errors.Join(errs...)
	}, nil
}

// newResource describes this process. Detectors are merged in order and later
// ones win, so the Config fallbacks come first and the standard
// OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES variables override them.
func newResource(ctx context.Context, serviceName string, attrs []attribute.KeyValue) (*resource.Resource, error) {
	opts := []resource.Option{
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	}
	if len(attrs) > 0 {
		opts = append(opts, resource.WithAttributes(attrs...))
	}
	if serviceName != "" {
		opts = append(opts, resource.WithAttributes(attribute.String("service.name", serviceName)))
	}
	opts = append(opts, resource.WithFromEnv())
	return resource.New(ctx, opts...)
}

// batchOptions maps non-zero Config fields to batch processor options. Zero
// fields are left to the SDK so its defaults and OTEL_BSP_* variables apply.
// WithBlocking is never set: a full queue drops spans instead of blocking.
func batchOptions(cfg Config) []sdktrace.BatchSpanProcessorOption {
	var opts []sdktrace.BatchSpanProcessorOption
	if cfg.MaxQueueSize > 0 {
		opts = append(opts, sdktrace.WithMaxQueueSize(cfg.MaxQueueSize))
	}
	if cfg.MaxExportBatchSize > 0 {
		opts = append(opts, sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatchSize))
	}
	if cfg.BatchTimeout > 0 {
		opts = append(opts, sdktrace.WithBatchTimeout(cfg.BatchTimeout))
	}
	if cfg.ExportTimeout > 0 {
		opts = append(opts, sdktrace.WithExportTimeout(cfg.ExportTimeout))
	}
	return opts
}
