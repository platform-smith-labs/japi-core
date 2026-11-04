package otel

import (
	"database/sql/driver"

	"github.com/platform-smith-labs/japi-core/v4/internal/sqltrace"
)

// WrapDriver returns d with database tracing, for drivers registered or
// opened by the caller (the db package's Config.Tracing covers its own
// connections). Register the result under a new name, or use WrapConnector:
//
//	sql.Register("postgres-traced", otel.WrapDriver(&pq.Driver{}))
//	db, err := sql.Open("postgres-traced", dsn)
//
// Each Exec, Query, prepared-statement execution, BEGIN, COMMIT and ROLLBACK
// gets a client span named after db.query.summary and carrying
// db.system.name ("postgresql"), db.operation.name (the first SQL keyword,
// upper-cased, or "_OTHER") and db.query.summary (the operation plus the
// target table when it is cheap to find, e.g. "SELECT users"). Statement
// arguments and statement text are never recorded. A failed operation sets the
// span status to error with a fixed description and error.type to the
// SQLSTATE when the driver exposes one; the driver's error text is not used,
// since it can quote row values.
//
// Each operation is also recorded in the db.client.operation.duration
// histogram (seconds) labelled only with db.system.name and
// db.operation.name. A query's span and duration cover the round trip that
// returns the first result, not iteration over the rows.
//
// The tracer and meter are the globals, resolved per operation.
func WrapDriver(d driver.Driver, opts ...SQLOption) driver.Driver {
	return sqltrace.WrapDriver(d, sqlOptions(opts))
}

// WrapConnector returns c with the tracing described at WrapDriver, for use
// with sql.OpenDB:
//
//	db := sql.OpenDB(otel.WrapConnector(connector))
func WrapConnector(c driver.Connector, opts ...SQLOption) driver.Connector {
	return sqltrace.WrapConnector(c, sqlOptions(opts))
}

// SQLOption configures WrapDriver and WrapConnector.
type SQLOption func(*sqltrace.Options)

// SQLRequireParentSpan starts a database span only when the operation's
// context already carries a valid span (an HTTP request, a traced job).
// Operations without one (pool upkeep, untraced background loops) are timed
// in db.client.operation.duration but start no trace of their own.
func SQLRequireParentSpan() SQLOption {
	return func(o *sqltrace.Options) { o.RequireParent = true }
}

func sqlOptions(opts []SQLOption) sqltrace.Options {
	var o sqltrace.Options
	for _, f := range opts {
		f(&o)
	}
	return o
}
