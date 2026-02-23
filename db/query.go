package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/georgysavva/scany/v2/sqlscan"
)

// Querier interface that both *sql.DB and *sql.Tx implement
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// WithTx executes a function within a database transaction
func WithTx[T any](db *sql.DB, fn func(*sql.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := db.Begin()
	if err != nil {
		return zero, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	result, err := fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return zero, fmt.Errorf("transaction failed: %v, rollback failed: %v", err, rbErr)
		}
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// QueryMany executes a query with positional parameters and uses automatic struct scanning
func QueryMany[T any](querier Querier, query string, args ...any) ([]T, error) {
	ctx := context.Background()
	rows, err := querier.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []T
	err = sqlscan.ScanAll(&results, rows)
	return results, err
}

// QueryOne executes a single row query with positional parameters and uses automatic struct scanning
func QueryOne[T any](querier Querier, query string, args ...any) (T, error) {
	var zero T

	slog.Debug("QueryOne executing",
		"query", query,
		"args", args,
	)

	ctx := context.Background()
	rows, err := querier.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Error("QueryOne failed",
			"query", query,
			"args", args,
			"error", err,
		)
		return zero, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Handle both pointer and non-pointer types
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		// For pointer types, allocate memory
		result := reflect.New(typ.Elem()).Interface().(T)
		err = sqlscan.ScanOne(result, rows)
		if err != nil {
			return zero, err
		}
		return result, err
	}

	// For non-pointer types, use the existing logic
	var result T
	err = sqlscan.ScanOne(&result, rows)
	return result, err
}

// Exec executes a query with positional parameters without returning results
func Exec(querier Querier, query string, args ...any) (sql.Result, error) {
	ctx := context.Background()
	result, err := querier.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}
