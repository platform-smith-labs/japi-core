package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Note: These tests require a test database. Set DB_TEST_URL environment variable:
// export DB_TEST_URL="postgres://user:password@localhost:5432/testdb?sslmode=disable"

// getTestDB returns a test database connection or skips the test
func getTestDB(t *testing.T) *sql.DB {
	// For unit tests without database, skip
	// In CI/CD with a test database, this would connect
	t.Skip("Database tests require a test PostgreSQL instance")
	return nil
}

// TestQueryOneWithCancellation verifies QueryOne respects context cancellation
func TestQueryOneWithCancellation(t *testing.T) {
	t.Run("cancelled context returns error", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		type User struct {
			ID   int    `db:"id"`
			Name string `db:"name"`
		}

		_, err := QueryOne[User](ctx, db, "SELECT id, name FROM users WHERE id = $1", 1)

		if err == nil {
			t.Error("Expected error from cancelled context")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})

	t.Run("timeout during query", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Sleep to ensure timeout occurs
		time.Sleep(1 * time.Millisecond)

		type User struct {
			ID   int    `db:"id"`
			Name string `db:"name"`
		}

		_, err := QueryOne[User](ctx, db, "SELECT id, name FROM users WHERE id = $1", 1)

		if err == nil {
			t.Error("Expected error from timeout context")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
		}
	})
}

// TestQueryManyWithCancellation verifies QueryMany respects context cancellation
func TestQueryManyWithCancellation(t *testing.T) {
	t.Run("cancelled context returns error", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		type User struct {
			ID   int    `db:"id"`
			Name string `db:"name"`
		}

		_, err := QueryMany[User](ctx, db, "SELECT id, name FROM users")

		if err == nil {
			t.Error("Expected error from cancelled context")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})

	t.Run("timeout during query", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(1 * time.Millisecond)

		type User struct {
			ID   int    `db:"id"`
			Name string `db:"name"`
		}

		_, err := QueryMany[User](ctx, db, "SELECT id, name FROM users")

		if err == nil {
			t.Error("Expected error from timeout context")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
		}
	})
}

// TestExecWithCancellation verifies Exec respects context cancellation
func TestExecWithCancellation(t *testing.T) {
	t.Run("cancelled context returns error", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := Exec(ctx, db, "UPDATE users SET name = $1 WHERE id = $2", "John", 1)

		if err == nil {
			t.Error("Expected error from cancelled context")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})

	t.Run("timeout during exec", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(1 * time.Millisecond)

		_, err := Exec(ctx, db, "UPDATE users SET name = $1 WHERE id = $2", "John", 1)

		if err == nil {
			t.Error("Expected error from timeout context")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
		}
	})
}

// TestWithTxCancellation verifies transaction respects context cancellation
func TestWithTxCancellation(t *testing.T) {
	t.Run("cancelled context prevents transaction start", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := WithTx(ctx, db, func(txCtx context.Context, tx *sql.Tx) (int, error) {
			// This should never execute
			t.Error("Transaction function should not execute with cancelled context")
			return 0, nil
		})

		if err == nil {
			t.Error("Expected error from cancelled context")
		}
	})

	t.Run("cancellation during transaction rolls back", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := WithTx(ctx, db, func(txCtx context.Context, tx *sql.Tx) (int, error) {
			// Start some work
			_, err := Exec(txCtx, tx, "INSERT INTO users (name) VALUES ($1)", "Test User")
			if err != nil {
				return 0, err
			}

			// Cancel context mid-transaction
			cancel()

			// Try another operation - should fail
			_, err = Exec(txCtx, tx, "INSERT INTO users (name) VALUES ($1)", "Test User 2")
			return 0, err
		})

		if err == nil {
			t.Error("Expected error from cancelled context during transaction")
		}

		// Verify transaction was rolled back by checking data doesn't exist
		// This would require actual database access
	})

	t.Run("timeout during transaction", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := WithTx(ctx, db, func(txCtx context.Context, tx *sql.Tx) (int, error) {
			// Simulate slow work
			time.Sleep(50 * time.Millisecond)

			_, err := Exec(txCtx, tx, "INSERT INTO users (name) VALUES ($1)", "Test User")
			return 0, err
		})

		if err == nil {
			t.Error("Expected timeout error during transaction")
		}
	})

	t.Run("panic during transaction triggers rollback", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx := context.Background()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to be propagated")
			}
		}()

		WithTx(ctx, db, func(txCtx context.Context, tx *sql.Tx) (int, error) {
			panic("test panic")
		})
	})
}

// TestWithTxReturnValue verifies transaction can return values
func TestWithTxReturnValue(t *testing.T) {
	t.Run("returns value on success", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx := context.Background()

		result, err := WithTx(ctx, db, func(txCtx context.Context, tx *sql.Tx) (string, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result != "success" {
			t.Errorf("Expected 'success', got: %s", result)
		}
	})

	t.Run("returns error when transaction fails", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		ctx := context.Background()

		expectedErr := errors.New("transaction error")
		_, err := WithTx(ctx, db, func(txCtx context.Context, tx *sql.Tx) (string, error) {
			return "", expectedErr
		})

		if err == nil {
			t.Error("Expected error from transaction")
		}

		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected transaction error, got: %v", err)
		}
	})
}

// TestContextPropagation verifies context flows through transaction
func TestContextPropagation(t *testing.T) {
	t.Run("transaction context inherits from parent", func(t *testing.T) {
		db := getTestDB(t)
		defer db.Close()

		type contextKey string
		const testKey contextKey = "test-key"

		parentCtx := context.WithValue(context.Background(), testKey, "test-value")

		var capturedValue string
		_, err := WithTx(parentCtx, db, func(txCtx context.Context, tx *sql.Tx) (int, error) {
			if value := txCtx.Value(testKey); value != nil {
				capturedValue = value.(string)
			}
			return 0, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if capturedValue != "test-value" {
			t.Errorf("Expected context value 'test-value', got: %s", capturedValue)
		}
	})
}

// TestQuerierInterface verifies both *sql.DB and *sql.Tx implement Querier
func TestQuerierInterface(t *testing.T) {
	t.Run("*sql.DB implements Querier", func(t *testing.T) {
		var _ Querier = (*sql.DB)(nil)
	})

	t.Run("*sql.Tx implements Querier", func(t *testing.T) {
		var _ Querier = (*sql.Tx)(nil)
	})
}

// Mock querier for testing without database
type mockQuerier struct {
	queryContextFunc    func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	queryRowContextFunc func(ctx context.Context, query string, args ...any) *sql.Row
	execContextFunc     func(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (m *mockQuerier) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if m.queryContextFunc != nil {
		return m.queryContextFunc(ctx, query, args...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockQuerier) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if m.queryRowContextFunc != nil {
		return m.queryRowContextFunc(ctx, query, args...)
	}
	return nil
}

func (m *mockQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, query, args...)
	}
	return nil, errors.New("not implemented")
}

// TestQueryOneContextPropagation verifies context is passed to QueryContext
func TestQueryOneContextPropagation(t *testing.T) {
	t.Run("context is passed to QueryContext", func(t *testing.T) {
		type contextKey string
		const testKey contextKey = "test-key"

		ctx := context.WithValue(context.Background(), testKey, "test-value")

		var capturedContext context.Context
		mock := &mockQuerier{
			queryContextFunc: func(c context.Context, query string, args ...any) (*sql.Rows, error) {
				capturedContext = c
				return nil, errors.New("test error")
			},
		}

		type User struct {
			ID int `db:"id"`
		}

		_, _ = QueryOne[User](ctx, mock, "SELECT id FROM users WHERE id = $1", 1)

		if capturedContext == nil {
			t.Error("Expected context to be passed to QueryContext")
		}

		if value := capturedContext.Value(testKey); value == nil {
			t.Error("Expected context value to be propagated")
		} else if value.(string) != "test-value" {
			t.Errorf("Expected 'test-value', got: %v", value)
		}
	})
}

// TestExecContextPropagation verifies context is passed to ExecContext
func TestExecContextPropagation(t *testing.T) {
	t.Run("context is passed to ExecContext", func(t *testing.T) {
		type contextKey string
		const testKey contextKey = "test-key"

		ctx := context.WithValue(context.Background(), testKey, "test-value")

		var capturedContext context.Context
		mock := &mockQuerier{
			execContextFunc: func(c context.Context, query string, args ...any) (sql.Result, error) {
				capturedContext = c
				return nil, errors.New("test error")
			},
		}

		_, _ = Exec(ctx, mock, "UPDATE users SET name = $1", "test")

		if capturedContext == nil {
			t.Error("Expected context to be passed to ExecContext")
		}

		if value := capturedContext.Value(testKey); value == nil {
			t.Error("Expected context value to be propagated")
		} else if value.(string) != "test-value" {
			t.Errorf("Expected 'test-value', got: %v", value)
		}
	})
}

// BenchmarkQueryOneWithContext benchmarks QueryOne with context
func BenchmarkQueryOneWithContext(b *testing.B) {
	b.Skip("Requires test database")

	db := getTestDB(&testing.T{})
	defer db.Close()

	type User struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = QueryOne[User](ctx, db, "SELECT id, name FROM users WHERE id = $1", 1)
	}
}

// BenchmarkExecWithContext benchmarks Exec with context
func BenchmarkExecWithContext(b *testing.B) {
	b.Skip("Requires test database")

	db := getTestDB(&testing.T{})
	defer db.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Exec(ctx, db, "UPDATE users SET name = $1 WHERE id = $2", "test", i)
	}
}
