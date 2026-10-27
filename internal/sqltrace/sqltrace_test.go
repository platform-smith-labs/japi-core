package sqltrace

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const marker = "SECRET-MARKER-123"

// fakeDriver is an in-memory driver: queries return one row with one column,
// and statements containing "fail" return an error quoting the arguments,
// the way real drivers quote row values.
type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return fakeConn{}, nil }

type fakeConn struct{}

func (fakeConn) Prepare(q string) (driver.Stmt, error) { return fakeStmt{q: q}, nil }
func (fakeConn) Close() error                          { return nil }
func (fakeConn) Begin() (driver.Tx, error)             { return fakeTx{}, nil }

func (fakeConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, "skip") {
		return nil, driver.ErrSkip
	}
	if err := failure(q, args); err != nil {
		return nil, err
	}
	return &fakeRows{}, nil
}

func (fakeConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(q, "skip") {
		return nil, driver.ErrSkip
	}
	if err := failure(q, args); err != nil {
		return nil, err
	}
	return driver.RowsAffected(1), nil
}

type sqlStateError struct{ msg string }

func (e sqlStateError) Error() string    { return e.msg }
func (e sqlStateError) SQLState() string { return "23505" }

func failure(q string, args []driver.NamedValue) error {
	if !strings.Contains(q, "fail") {
		return nil
	}
	msg := "duplicate key: " + q
	for _, a := range args {
		msg += " " + a.Value.(string)
	}
	return sqlStateError{msg: msg}
}

type fakeStmt struct{ q string }

func (s fakeStmt) Close() error  { return nil }
func (s fakeStmt) NumInput() int { return -1 }
func (s fakeStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}
func (s fakeStmt) Query([]driver.Value) (driver.Rows, error) { return &fakeRows{}, nil }

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

type fakeRows struct{ done bool }

func (*fakeRows) Columns() []string { return []string{"n"} }
func (*fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = int64(1)
	return nil
}

type fakeConnector struct{}

func (fakeConnector) Connect(context.Context) (driver.Conn, error) { return fakeConn{}, nil }
func (fakeConnector) Driver() driver.Driver                        { return fakeDriver{} }

func newRecorders(t *testing.T) (Options, *tracetest.SpanRecorder, *sdkmetric.ManualReader) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	return Options{TracerProvider: tp, MeterProvider: mp}, sr, reader
}

// assertNoMarker fails if the marker appears anywhere in a span.
func assertNoMarker(t *testing.T, s sdktrace.ReadOnlySpan) {
	t.Helper()
	check := func(where, v string) {
		if strings.Contains(v, marker) || strings.Contains(v, "secret_table_text") {
			t.Errorf("span %q leaks the marker in %s: %q", s.Name(), where, v)
		}
	}
	check("name", s.Name())
	check("status", s.Status().Description)
	for _, kv := range s.Attributes() {
		check("attribute "+string(kv.Key), kv.Value.Emit())
	}
	for _, e := range s.Events() {
		check("event name", e.Name)
		for _, kv := range e.Attributes {
			check("event attribute "+string(kv.Key), kv.Value.Emit())
		}
	}
}

func attrs(s sdktrace.ReadOnlySpan) map[attribute.Key]string {
	m := map[attribute.Key]string{}
	for _, kv := range s.Attributes() {
		m[kv.Key] = kv.Value.Emit()
	}
	return m
}

func TestWrapConnector_SpanHasNoArgumentsOrQueryText(t *testing.T) {
	opts, sr, reader := newRecorders(t)
	db := sql.OpenDB(WrapConnector(fakeConnector{}, opts))
	defer db.Close()
	ctx := context.Background()

	var n int
	if err := db.QueryRowContext(ctx, "SELECT n FROM users WHERE note = 'secret_table_text' AND email = $1", marker).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO fail_table (v) VALUES ($1)", marker); err == nil {
		t.Fatal("expected the failing exec to fail")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.PrepareContext(ctx, "UPDATE accounts SET v = $1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stmt.ExecContext(ctx, marker); err != nil {
		t.Fatal(err)
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	spans := sr.Ended()
	var names []string
	for _, s := range spans {
		names = append(names, s.Name())
		assertNoMarker(t, s)
		a := attrs(s)
		if a["db.system.name"] != "postgresql" || a["db.operation.name"] == "" {
			t.Errorf("span %q attributes = %v", s.Name(), a)
		}
	}
	want := []string{"SELECT users", "INSERT fail_table", "BEGIN", "UPDATE accounts", "COMMIT"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("span names = %v, want %v", names, want)
	}
	failed := spans[1]
	if failed.Status().Code != codes.Error || attrs(failed)["error.type"] != "23505" {
		t.Errorf("failed span status = %v, error.type = %q", failed.Status(), attrs(failed)["error.type"])
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	var points uint64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "db.client.operation.duration" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
				points += dp.Count
				for _, kv := range dp.Attributes.ToSlice() {
					if kv.Key != "db.system.name" && kv.Key != "db.operation.name" {
						t.Errorf("duration label %s not allowed", kv.Key)
					}
				}
			}
		}
	}
	if points != uint64(len(want)) {
		t.Errorf("duration measurements = %d, want %d", points, len(want))
	}
}

func TestWrapDriver_OpenAndConnectorAreTraced(t *testing.T) {
	opts, sr, _ := newRecorders(t)
	d := WrapDriver(fakeDriver{}, opts)
	connector, err := d.(driver.DriverContext).OpenConnector("dsn")
	if err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	if _, err := db.Exec("DELETE FROM sessions WHERE id = $1", marker); err != nil {
		t.Fatal(err)
	}
	spans := sr.Ended()
	if len(spans) != 1 || spans[0].Name() != "DELETE sessions" {
		t.Fatalf("spans = %v, want one DELETE sessions", spans)
	}
	assertNoMarker(t, spans[0])
}

func TestUnwrapped_NoSpans(t *testing.T) {
	_, sr, _ := newRecorders(t)
	db := sql.OpenDB(fakeConnector{})
	defer db.Close()
	if _, err := db.Exec("SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if n := len(sr.Ended()); n != 0 {
		t.Errorf("unwrapped driver produced %d spans", n)
	}
}

func TestErrorType(t *testing.T) {
	if got := errorType(errors.New("x")); got != "_OTHER" {
		t.Errorf("errorType(plain) = %q", got)
	}
	if got := errorType(sqlStateError{}); got != "23505" {
		t.Errorf("errorType(sqlstate) = %q", got)
	}
}

// TestErrSkip_NotCountedAsFailureOrTwice verifies that when the driver answers
// ExecContext/QueryContext with driver.ErrSkip (database/sql then retries
// through Prepare), the skipped attempt has no error status and records no
// duration, so each statement is measured once.
func TestErrSkip_NotCountedAsFailureOrTwice(t *testing.T) {
	opts, sr, reader := newRecorders(t)
	db := sql.OpenDB(WrapConnector(fakeConnector{}, opts))
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "UPDATE skip_t SET a = 1"); err != nil {
		t.Fatalf("exec: %v", err)
	}
	rows, err := db.QueryContext(ctx, "SELECT n FROM skip_t")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	rows.Close()

	for _, s := range sr.Ended() {
		if s.Status().Code == codes.Error {
			t.Errorf("span %q has error status after ErrSkip", s.Name())
		}
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	var points uint64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "db.client.operation.duration" {
				for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
					points += dp.Count
				}
			}
		}
	}
	if points != 2 {
		t.Errorf("duration measurements = %d, want 2 (one per statement)", points)
	}
}
