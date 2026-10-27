package sqltrace

import (
	"context"
	"database/sql/driver"
	"errors"
)

// wrappedConn implements every optional connection interface database/sql
// looks for, delegating to the wrapped connection or reproducing
// database/sql's own fallback when the wrapped connection lacks one.
type wrappedConn struct {
	c driver.Conn
	t *tracer
}

var (
	_ driver.Conn               = (*wrappedConn)(nil)
	_ driver.ConnPrepareContext = (*wrappedConn)(nil)
	_ driver.ConnBeginTx        = (*wrappedConn)(nil)
	_ driver.ExecerContext      = (*wrappedConn)(nil)
	_ driver.QueryerContext     = (*wrappedConn)(nil)
	_ driver.Pinger             = (*wrappedConn)(nil)
	_ driver.SessionResetter    = (*wrappedConn)(nil)
	_ driver.Validator          = (*wrappedConn)(nil)
	_ driver.NamedValueChecker  = (*wrappedConn)(nil)
)

func (c *wrappedConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *wrappedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	var (
		s   driver.Stmt
		err error
	)
	if pc, ok := c.c.(driver.ConnPrepareContext); ok {
		s, err = pc.PrepareContext(ctx, query)
	} else {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		s, err = c.c.Prepare(query)
	}
	if err != nil {
		return nil, err
	}
	operation, summary := summarize(query)
	return &wrappedStmt{s: s, conn: c, operation: operation, summary: summary}, nil
}

func (c *wrappedConn) Close() error { return c.c.Close() }

// Begin is required by driver.Conn; database/sql calls BeginTx instead.
func (c *wrappedConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *wrappedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	o := c.t.begin(ctx, "BEGIN", "BEGIN")
	var (
		tx  driver.Tx
		err error
	)
	if bt, ok := c.c.(driver.ConnBeginTx); ok {
		tx, err = bt.BeginTx(o.ctx, opts)
	} else if opts.Isolation != 0 || opts.ReadOnly {
		err = errors.New("sqltrace: driver does not support non-default transaction options")
	} else if err = ctx.Err(); err == nil {
		tx, err = c.c.Begin() //nolint:staticcheck // fallback for drivers without ConnBeginTx
	}
	o.end(err)
	if err != nil {
		return nil, err
	}
	return &wrappedTx{tx: tx, t: c.t, ctx: ctx}, nil
}

func (c *wrappedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	ec, ok := c.c.(driver.ExecerContext)
	if !ok {
		// database/sql falls back to Prepare + Exec, which is traced.
		return nil, driver.ErrSkip
	}
	o := c.t.beginQuery(ctx, query)
	res, err := ec.ExecContext(o.ctx, query, args)
	o.end(err)
	return res, err
}

func (c *wrappedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	qc, ok := c.c.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	o := c.t.beginQuery(ctx, query)
	rows, err := qc.QueryContext(o.ctx, query, args)
	o.end(err)
	return rows, err
}

func (c *wrappedConn) Ping(ctx context.Context) error {
	if p, ok := c.c.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (c *wrappedConn) ResetSession(ctx context.Context) error {
	if r, ok := c.c.(driver.SessionResetter); ok {
		return r.ResetSession(ctx)
	}
	return nil
}

func (c *wrappedConn) IsValid() bool {
	if v, ok := c.c.(driver.Validator); ok {
		return v.IsValid()
	}
	return true
}

func (c *wrappedConn) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := c.c.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

// wrappedTx traces COMMIT and ROLLBACK as children of the context the
// transaction was begun with (driver.Tx methods take no context).
type wrappedTx struct {
	tx  driver.Tx
	t   *tracer
	ctx context.Context
}

func (tx *wrappedTx) Commit() error {
	o := tx.t.begin(tx.ctx, "COMMIT", "COMMIT")
	err := tx.tx.Commit()
	o.end(err)
	return err
}

func (tx *wrappedTx) Rollback() error {
	o := tx.t.begin(tx.ctx, "ROLLBACK", "ROLLBACK")
	err := tx.tx.Rollback()
	o.end(err)
	return err
}

// wrappedStmt traces each execution of a prepared statement. The summary is
// derived once, at prepare time; the statement text is not kept.
type wrappedStmt struct {
	s         driver.Stmt
	conn      *wrappedConn
	operation string
	summary   string
}

var (
	_ driver.StmtExecContext   = (*wrappedStmt)(nil)
	_ driver.StmtQueryContext  = (*wrappedStmt)(nil)
	_ driver.NamedValueChecker = (*wrappedStmt)(nil)
)

func (s *wrappedStmt) Close() error  { return s.s.Close() }
func (s *wrappedStmt) NumInput() int { return s.s.NumInput() }

func (s *wrappedStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), namedValues(args))
}

func (s *wrappedStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), namedValues(args))
}

func (s *wrappedStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	o := s.conn.t.begin(ctx, s.operation, s.summary)
	var (
		res driver.Result
		err error
	)
	if ec, ok := s.s.(driver.StmtExecContext); ok {
		res, err = ec.ExecContext(o.ctx, args)
	} else {
		var vals []driver.Value
		if vals, err = plainValues(args); err == nil {
			if err = ctx.Err(); err == nil {
				res, err = s.s.Exec(vals) //nolint:staticcheck // fallback for drivers without StmtExecContext
			}
		}
	}
	o.end(err)
	return res, err
}

func (s *wrappedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	o := s.conn.t.begin(ctx, s.operation, s.summary)
	var (
		rows driver.Rows
		err  error
	)
	if qc, ok := s.s.(driver.StmtQueryContext); ok {
		rows, err = qc.QueryContext(o.ctx, args)
	} else {
		var vals []driver.Value
		if vals, err = plainValues(args); err == nil {
			if err = ctx.Err(); err == nil {
				rows, err = s.s.Query(vals) //nolint:staticcheck // fallback for drivers without StmtQueryContext
			}
		}
	}
	o.end(err)
	return rows, err
}

// CheckNamedValue consults the statement's checker, then the connection's:
// database/sql stops at the statement's when the statement has one, so the
// connection's must be reached from here.
func (s *wrappedStmt) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := s.s.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return s.conn.CheckNamedValue(nv)
}

func namedValues(args []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, len(args))
	for i, v := range args {
		out[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return out
}

func plainValues(args []driver.NamedValue) ([]driver.Value, error) {
	out := make([]driver.Value, len(args))
	for i, a := range args {
		if a.Name != "" {
			return nil, errors.New("sqltrace: driver does not support named parameters")
		}
		out[i] = a.Value
	}
	return out, nil
}
