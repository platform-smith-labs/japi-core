// Package sqltrace wraps a database/sql driver so that every statement is
// traced and timed through OpenTelemetry, without ever recording statement
// arguments or the statement text.
//
// It depends only on the OpenTelemetry API, so packages that offer tracing as
// an option do not link the SDK.
package sqltrace

import (
	"context"
	"database/sql/driver"
	"errors"
	"time"

	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/platform-smith-labs/japi-core/v4/otel/sql"

// errorDescription is the span status description of a failed operation.
// The driver's error text is never used: it may quote values from the
// statement or the affected row.
const errorDescription = "database operation failed"

// durationBuckets are the bucket boundaries (seconds) the database semantic
// conventions advise for db.client.operation.duration.
var durationBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10}

// Options selects the providers. Nil fields use the globals, resolved on
// every operation so a later SDK setup takes effect.
type Options struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// tracer holds the instrumentation state shared by everything a wrapped
// driver hands out.
type tracer struct {
	opts Options
}

func (t *tracer) traceProvider() trace.TracerProvider {
	if t.opts.TracerProvider != nil {
		return t.opts.TracerProvider
	}
	return otelapi.GetTracerProvider()
}

func (t *tracer) duration() metric.Float64Histogram {
	mp := t.opts.MeterProvider
	if mp == nil {
		mp = otelapi.GetMeterProvider()
	}
	h, err := mp.Meter(instrumentationName).Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of database client operations."),
		metric.WithExplicitBucketBoundaries(durationBuckets...),
	)
	if err != nil {
		otelapi.Handle(err)
		h, _ = metricnoop.NewMeterProvider().Meter("").Float64Histogram("")
	}
	return h
}

// op is one traced database operation.
type op struct {
	t         *tracer
	ctx       context.Context
	span      trace.Span
	operation string
	start     time.Time
}

// begin starts a client span named after summary. Only the operation name and
// summary are recorded; the statement is never passed in.
func (t *tracer) begin(ctx context.Context, operation, summary string) *op {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := t.traceProvider().Tracer(instrumentationName).Start(ctx, summary,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", operation),
			attribute.String("db.query.summary", summary),
		),
	)
	return &op{t: t, ctx: ctx, span: span, operation: operation, start: time.Now()}
}

// beginQuery starts an op for a statement.
func (t *tracer) beginQuery(ctx context.Context, query string) *op {
	operation, summary := summarize(query)
	return t.begin(ctx, operation, summary)
}

// end records the duration and ends the span. err is reduced to an
// error.type: the SQLSTATE when the driver error exposes one, else "_OTHER".
//
// driver.ErrSkip is not a failure: database/sql retries the statement another
// way (e.g. prepare and execute), and that attempt is traced on its own. The
// span is ended without an error status and no duration is recorded, so the
// operation is not counted twice.
func (o *op) end(err error) {
	if errors.Is(err, driver.ErrSkip) {
		o.span.End()
		return
	}
	o.t.duration().Record(o.ctx, time.Since(o.start).Seconds(), metric.WithAttributeSet(attribute.NewSet(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", o.operation),
	)))
	if err != nil {
		o.span.SetAttributes(attribute.String("error.type", errorType(err)))
		o.span.SetStatus(codes.Error, errorDescription)
	}
	o.span.End()
}

// errorType returns the SQLSTATE of err (pgx and lib/pq errors expose one
// through SQLState()) or "_OTHER".
func errorType(err error) string {
	var state interface{ SQLState() string }
	if errors.As(err, &state) {
		if s := state.SQLState(); len(s) == 5 {
			return s
		}
	}
	return "_OTHER"
}

// WrapDriver returns d with tracing. The returned driver also implements
// driver.DriverContext; its connectors are traced too.
func WrapDriver(d driver.Driver, opts Options) driver.Driver {
	return &wrappedDriver{d: d, t: &tracer{opts: opts}}
}

// WrapConnector returns c with tracing, for use with sql.OpenDB.
func WrapConnector(c driver.Connector, opts Options) driver.Connector {
	t := &tracer{opts: opts}
	return &wrappedConnector{c: c, t: t, d: &wrappedDriver{d: c.Driver(), t: t}}
}

// IsWrapped reports whether v (a driver, connector or connection) was
// returned by this package.
func IsWrapped(v any) bool {
	switch v.(type) {
	case *wrappedDriver, *wrappedConnector, *wrappedConn:
		return true
	}
	return false
}

type wrappedDriver struct {
	d driver.Driver
	t *tracer
}

func (d *wrappedDriver) Open(name string) (driver.Conn, error) {
	c, err := d.d.Open(name)
	if err != nil {
		return nil, err
	}
	return &wrappedConn{c: c, t: d.t}, nil
}

func (d *wrappedDriver) OpenConnector(name string) (driver.Connector, error) {
	if dc, ok := d.d.(driver.DriverContext); ok {
		c, err := dc.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return &wrappedConnector{c: c, t: d.t, d: d}, nil
	}
	return &dsnConnector{name: name, d: d}, nil
}

type wrappedConnector struct {
	c driver.Connector
	t *tracer
	d *wrappedDriver
}

func (c *wrappedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.c.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &wrappedConn{c: conn, t: c.t}, nil
}

func (c *wrappedConnector) Driver() driver.Driver { return c.d }

// dsnConnector mirrors database/sql's fallback for drivers without
// DriverContext.
type dsnConnector struct {
	name string
	d    *wrappedDriver
}

func (c *dsnConnector) Connect(context.Context) (driver.Conn, error) { return c.d.Open(c.name) }
func (c *dsnConnector) Driver() driver.Driver                        { return c.d }
