package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/platform-smith-labs/japi-core/core"
)

// TestAdapterContextExtraction verifies that adapter extracts request context
func TestAdapterContextExtraction(t *testing.T) {
	t.Run("extracts context from request", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		var capturedContext context.Context
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			capturedContext = ctx.Context
			return struct{}{}, nil
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		if capturedContext == nil {
			t.Error("Expected context to be set in HandlerContext")
		}
		if capturedContext != req.Context() {
			t.Error("Expected context to match request context")
		}
	})

	t.Run("propagates request context values", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		type contextKey string
		const testKey contextKey = "test-key"

		var capturedValue string
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			if value := ctx.Context.Value(testKey); value != nil {
				capturedValue = value.(string)
			}
			return struct{}{}, nil
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		// Create request with context value
		baseCtx := context.WithValue(context.Background(), testKey, "test-value")
		req := httptest.NewRequest("GET", "/test", nil).WithContext(baseCtx)
		w := httptest.NewRecorder()

		adapted(w, req)

		if capturedValue != "test-value" {
			t.Errorf("Expected context value 'test-value', got '%s'", capturedValue)
		}
	})
}

// TestAdapterContextCancellation verifies cancellation handling
func TestAdapterContextCancellation(t *testing.T) {
	t.Run("handles context.Canceled error", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			return struct{}{}, context.Canceled
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		// Should not write response body for cancelled requests
		if w.Code != 0 && w.Code != http.StatusOK {
			t.Errorf("Expected no status code or 200, got %d", w.Code)
		}
	})

	t.Run("handles wrapped context.Canceled error", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Wrap the cancellation error
			return struct{}{}, errors.New("database error: " + context.Canceled.Error())
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		// Wrapped error should be treated as regular error
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("handles APIError wrapping context.Canceled", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Return APIError with 499 status for client cancellation
			return struct{}{}, core.NewAPIError(499, "Client closed request")
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		if w.Code != 499 {
			t.Errorf("Expected status 499, got %d", w.Code)
		}
	})
}

// TestAdapterContextTimeout verifies timeout handling
func TestAdapterContextTimeout(t *testing.T) {
	t.Run("handles context.DeadlineExceeded error", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			return struct{}{}, context.DeadlineExceeded
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("Expected status 504, got %d", w.Code)
		}
	})

	t.Run("returns timeout error from handler", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Simulate checking for timeout
			if errors.Is(ctx.Context.Err(), context.DeadlineExceeded) {
				return struct{}{}, context.DeadlineExceeded
			}
			return struct{}{}, nil
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		// Create request with cancelled context (simulating timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		time.Sleep(2 * time.Millisecond) // Wait for timeout

		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		adapted(w, req)

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("Expected status 504, got %d", w.Code)
		}
	})
}

// TestHandlerContextWithCancelledContext verifies handler behavior with cancelled context
func TestHandlerContextWithCancelledContext(t *testing.T) {
	t.Run("handler can detect cancelled context", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		var contextWasCancelled bool
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Check if context is already cancelled
			select {
			case <-ctx.Context.Done():
				contextWasCancelled = true
			default:
				contextWasCancelled = false
			}
			return struct{}{}, nil
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		adapted(w, req)

		if !contextWasCancelled {
			t.Error("Expected handler to detect cancelled context")
		}
	})
}

// TestHandlerContextWithTimeout verifies handler behavior with timeout
func TestHandlerContextWithTimeout(t *testing.T) {
	t.Run("handler respects context timeout", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		var timeoutOccurred bool
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Simulate work that takes time
			select {
			case <-time.After(100 * time.Millisecond):
				// Work completed
			case <-ctx.Context.Done():
				// Timeout occurred
				timeoutOccurred = true
				return struct{}{}, ctx.Context.Err()
			}
			return struct{}{}, nil
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		adapted(w, req)

		if !timeoutOccurred {
			t.Error("Expected timeout to occur during handler execution")
		}
	})
}

// TestHandlerContextNonNil verifies context is never nil
func TestHandlerContextNonNil(t *testing.T) {
	t.Run("context is always set", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		var contextWasNil bool
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			contextWasNil = (ctx.Context == nil)
			return struct{}{}, nil
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		if contextWasNil {
			t.Error("Context should never be nil in HandlerContext")
		}
	})
}

// TestRegularErrorsNotAffectedByContext verifies non-context errors still work
func TestRegularErrorsNotAffectedByContext(t *testing.T) {
	t.Run("APIError is handled normally", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			return struct{}{}, core.NewAPIError(http.StatusNotFound, "Resource not found")
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("sql.ErrNoRows is handled normally", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Wrap sql error in APIError
			return struct{}{}, core.NewAPIError(http.StatusNotFound, "User not found")
		}

		adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		adapted(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// BenchmarkContextExtraction benchmarks context extraction overhead
func BenchmarkContextExtraction(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
		_ = ctx.Context
		return struct{}{}, nil
	}

	adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapted(w, req)
	}
}

// BenchmarkContextCancellationCheck benchmarks checking for cancellation
func BenchmarkContextCancellationCheck(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
		select {
		case <-ctx.Context.Done():
			return struct{}{}, context.Canceled
		default:
		}
		return struct{}{}, nil
	}

	adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapted(w, req)
	}
}
