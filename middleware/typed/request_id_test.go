package typed

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/platform-smith-labs/japi-core/handler"
	httpMiddleware "github.com/platform-smith-labs/japi-core/middleware/http"
)

// TestWithRequestID_EnrichesContext verifies request ID is added to HandlerContext
func TestWithRequestID_EnrichesContext(t *testing.T) {
	t.Run("adds request ID to HandlerContext", func(t *testing.T) {
		expectedRequestID := "test-request-id-123"

		// Create test handler that checks context
		testHandler := func(ctx handler.HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Verify request ID is in context
			if !ctx.RequestID.HasValue() {
				t.Error("Expected RequestID to have value, got no value")
			}

			requestID, err := ctx.RequestID.Value()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if requestID != expectedRequestID {
				t.Errorf("Expected request ID %s, got %s", expectedRequestID, requestID)
			}

			return struct{}{}, nil
		}

		// Apply middleware - types inferred from testHandler
		wrappedHandler := WithRequestID(testHandler)

		// Create request with request ID in context
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), httpMiddleware.RequestIDContextKey, expectedRequestID)
		req = req.WithContext(ctx)

		// Create HandlerContext
		handlerCtx := handler.HandlerContext[struct{}, struct{}]{
			Context: req.Context(),
			Logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		}

		// Execute handler
		rec := httptest.NewRecorder()
		_, err := wrappedHandler(handlerCtx, rec, req)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestWithRequestID_EmptyWhenNoContext verifies behavior when no request ID in context
func TestWithRequestID_EmptyWhenNoContext(t *testing.T) {
	t.Run("leaves RequestID invalid when no request ID in context", func(t *testing.T) {
		// Create test handler that checks context
		testHandler := func(ctx handler.HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Verify request ID is not set
			if ctx.RequestID.HasValue() {
				value, _ := ctx.RequestID.Value()
				t.Errorf("Expected RequestID to have no value, got value: %s", value)
			}

			return struct{}{}, nil
		}

		// Apply middleware - types inferred from testHandler
		wrappedHandler := WithRequestID(testHandler)

		// Create request WITHOUT request ID in context
		req := httptest.NewRequest("GET", "/test", nil)

		// Create HandlerContext
		handlerCtx := handler.HandlerContext[struct{}, struct{}]{
			Context: req.Context(),
			Logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		}

		// Execute handler
		rec := httptest.NewRecorder()
		_, err := wrappedHandler(handlerCtx, rec, req)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestWithRequestID_EnrichesLogger verifies logger is enriched with request ID
func TestWithRequestID_EnrichesLogger(t *testing.T) {
	t.Run("enriches logger with request_id field", func(t *testing.T) {
		expectedRequestID := "test-request-id-456"

		// Track if logger was enriched (we'll check by seeing if the logger reference changed)
		var originalLogger, enrichedLogger *slog.Logger

		// Create test handler that captures logger
		testHandler := func(ctx handler.HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			enrichedLogger = ctx.Logger
			return struct{}{}, nil
		}

		// Apply middleware - types inferred from testHandler
		wrappedHandler := WithRequestID(testHandler)

		// Create request with request ID in context
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), httpMiddleware.RequestIDContextKey, expectedRequestID)
		req = req.WithContext(ctx)

		// Create HandlerContext with original logger
		originalLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
		handlerCtx := handler.HandlerContext[struct{}, struct{}]{
			Context: req.Context(),
			Logger:  originalLogger,
		}

		// Execute handler
		rec := httptest.NewRecorder()
		_, err := wrappedHandler(handlerCtx, rec, req)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify logger was enriched (different instance)
		if enrichedLogger == originalLogger {
			t.Error("Expected logger to be enriched (new instance), got same instance")
		}

		if enrichedLogger == nil {
			t.Error("Expected enriched logger, got nil")
		}
	})
}

// TestWithRequestID_PreservesOtherContext verifies other context fields are preserved
func TestWithRequestID_PreservesOtherContext(t *testing.T) {
	t.Run("preserves other HandlerContext fields", func(t *testing.T) {
		expectedRequestID := "test-request-id-789"

		// Create test handler that checks all context fields
		testHandler := func(ctx handler.HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			// Verify request ID is set
			requestID, err := ctx.RequestID.Value()
			if !ctx.RequestID.HasValue() || err != nil || requestID != expectedRequestID {
				t.Error("Expected request ID to be set correctly")
			}

			// Verify logger is still present
			if ctx.Logger == nil {
				t.Error("Expected logger to be preserved")
			}

			// Verify DB is still present (even if nil in test)
			// Just checking the field exists

			return struct{}{}, nil
		}

		// Apply middleware - types inferred from testHandler
		wrappedHandler := WithRequestID(testHandler)

		// Create request with request ID in context
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), httpMiddleware.RequestIDContextKey, expectedRequestID)
		req = req.WithContext(ctx)

		// Create HandlerContext with logger
		handlerCtx := handler.HandlerContext[struct{}, struct{}]{
			Context: req.Context(),
			Logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
			DB:      nil, // Would normally be a real DB
		}

		// Execute handler
		rec := httptest.NewRecorder()
		_, err := wrappedHandler(handlerCtx, rec, req)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestWithRequestID_Integration verifies integration with HTTP middleware
func TestWithRequestID_Integration(t *testing.T) {
	t.Run("works with http.WithRequestID middleware", func(t *testing.T) {
		var capturedRequestID string

		// Create typed handler
		testHandler := func(ctx handler.HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			if ctx.RequestID.HasValue() {
				value, err := ctx.RequestID.Value()
				if err == nil {
					capturedRequestID = value
				}
			}
			return struct{}{}, nil
		}

		// Apply typed middleware - types inferred from testHandler
		typedHandler := WithRequestID(testHandler)

		// Wrap in HTTP middleware
		httpHandler := httpMiddleware.WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate what AdaptHandler would do
			handlerCtx := handler.HandlerContext[struct{}, struct{}]{
				Context: r.Context(),
				Logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
			}
			typedHandler(handlerCtx, w, r)
		}))

		// Create request
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Execute full middleware chain
		httpHandler.ServeHTTP(rec, req)

		// Verify request ID was propagated through both middleware layers
		if capturedRequestID == "" {
			t.Error("Expected request ID to be captured, got empty string")
		}

		// Verify response header has same request ID
		responseRequestID := rec.Header().Get(httpMiddleware.RequestIDHeader)
		if responseRequestID != capturedRequestID {
			t.Errorf("Expected response request ID %s to match captured %s", responseRequestID, capturedRequestID)
		}
	})
}

// TestWithRequestID_TypeInference verifies that type parameters can be inferred in MakeHandler
func TestWithRequestID_TypeInference(t *testing.T) {
	t.Run("infers types when used in MakeHandler", func(t *testing.T) {
		type TestParams struct {
			ID string `path:"id"`
		}
		type TestBody struct {
			Name string `json:"name"`
		}
		type TestResponse struct {
			Success bool `json:"success"`
		}

		// Create base handler with concrete types
		testHandler := func(ctx handler.HandlerContext[TestParams, TestBody], w http.ResponseWriter, r *http.Request) (TestResponse, error) {
			// Verify request ID was set by middleware
			if ctx.RequestID.HasValue() {
				return TestResponse{Success: true}, nil
			}
			return TestResponse{Success: false}, nil
		}

		// Create a test registry
		reg := handler.NewRegistry()

		// Use MakeHandler with WithRequestID - type inference works here!
		// Go infers types from testHandler signature
		composedHandler := handler.MakeHandler(
			reg,
			handler.RouteInfo{Method: "GET", Path: "/test"},
			testHandler,
			WithRequestID, // ← No explicit types or () needed! Type inference works!
		)

		// Verify it compiled and works
		if composedHandler == nil {
			t.Fatal("Expected composed handler to be non-nil")
		}

		// Verify the route was registered
		routes := reg.GetRoutes()
		if len(routes) != 1 {
			t.Errorf("Expected 1 route registered, got %d", len(routes))
		}

		// This test proves that type inference works in the real use case:
		// Inside MakeHandler, Go can infer the middleware types from the baseHandler
		t.Log("✓ Type inference works! No explicit type parameters needed in MakeHandler")
	})
}

// BenchmarkWithRequestID benchmarks typed request ID middleware
func BenchmarkWithRequestID(b *testing.B) {
	testHandler := func(ctx handler.HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
		return struct{}{}, nil
	}

	wrappedHandler := WithRequestID(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), httpMiddleware.RequestIDContextKey, "benchmark-request-id")
	req = req.WithContext(ctx)

	handlerCtx := handler.HandlerContext[struct{}, struct{}]{
		Context: req.Context(),
		Logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}

	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrappedHandler(handlerCtx, rec, req)
	}
}
