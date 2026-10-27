package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// sqlStateError mimics a driver error exposing SQLState().
type sqlStateError struct{ code string }

func (e *sqlStateError) Error() string    { return "duplicate key value violates unique constraint" }
func (e *sqlStateError) SQLState() string { return e.code }

// failingQuerier fails every query with err.
type failingQuerier struct{ err error }

func (q failingQuerier) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, q.err
}
func (q failingQuerier) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }
func (q failingQuerier) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, q.err
}

func captureDefaultLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func TestQueryOne_ErrorLogOmitsArgs(t *testing.T) {
	const marker = "ARG-MARKER-a@example.com"
	buf := captureDefaultLog(t)

	q := failingQuerier{err: fmt.Errorf("wrapped: %w", &sqlStateError{code: "23505"})}
	_, err := QueryOne[struct{ ID int }](context.Background(), q, "SELECT id FROM t WHERE email = $1", marker)
	if err == nil {
		t.Fatal("want error")
	}

	out := buf.String()
	if strings.Contains(out, marker) {
		t.Errorf("query argument logged:\n%s", out)
	}
	if strings.Contains(out, `"args"`) {
		t.Errorf("args key still logged:\n%s", out)
	}

	var failed map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("decode %q: %v", l, err)
		}
		if m["msg"] == "QueryOne failed" {
			failed = m
		}
	}
	if failed == nil {
		t.Fatalf("no failure line:\n%s", out)
	}
	if failed["db.response.status_code"] != "23505" {
		t.Errorf("db.response.status_code = %v, want 23505", failed["db.response.status_code"])
	}
}

func TestQueryOne_NoSQLStateKeyForPlainErrors(t *testing.T) {
	buf := captureDefaultLog(t)
	_, _ = QueryOne[int](context.Background(), failingQuerier{err: errors.New("conn refused")}, "SELECT 1")
	if strings.Contains(buf.String(), "db.response.status_code") {
		t.Errorf("status code logged for an error without SQLSTATE:\n%s", buf.String())
	}
}

// echoingError mimics a driver error whose message quotes a bound value.
type echoingError struct{ value string }

func (e *echoingError) Error() string {
	return `invalid input syntax for type uuid: "` + e.value + `"`
}
func (e *echoingError) SQLState() string { return "22P02" }

func TestQueryOne_ErrorLogOmitsErrorText(t *testing.T) {
	const marker = "ECHO-MARKER-not-a-uuid"
	buf := captureDefaultLog(t)

	_, err := QueryOne[int](context.Background(), failingQuerier{err: &echoingError{value: marker}},
		"SELECT 1 WHERE id = $1", marker)
	if err == nil {
		t.Fatal("want error")
	}
	out := buf.String()
	if strings.Contains(out, marker) {
		t.Errorf("error text echoing a bound value was logged:\n%s", out)
	}
	if !strings.Contains(out, `"error.type":"*db.echoingError"`) {
		t.Errorf("error.type missing:\n%s", out)
	}
	if !strings.Contains(out, `"db.response.status_code":"22P02"`) {
		t.Errorf("db.response.status_code missing:\n%s", out)
	}
}
