package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestWithRequestID_GeneratesNewID verifies that a new request ID is generated when none exists
func TestWithRequestID_GeneratesNewID(t *testing.T) {
	t.Run("generates UUID when no request ID header present", func(t *testing.T) {
		// Create test handler that captures the request
		var capturedRequestID string
		handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRequestID = GetRequestID(r)
			w.WriteHeader(http.StatusOK)
		}))

		// Create request without X-Request-ID header
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Execute handler
		handler.ServeHTTP(rec, req)

		// Verify request ID was generated and is a valid UUID
		if capturedRequestID == "" {
			t.Error("Expected request ID to be generated, got empty string")
		}

		// Verify it's a valid UUID
		if _, err := uuid.Parse(capturedRequestID); err != nil {
			t.Errorf("Expected valid UUID, got %s: %v", capturedRequestID, err)
		}

		// Verify response header contains request ID
		responseRequestID := rec.Header().Get(RequestIDHeader)
		if responseRequestID != capturedRequestID {
			t.Errorf("Expected response header to match context request ID, got %s != %s", responseRequestID, capturedRequestID)
		}
	})
}

// TestWithRequestID_PropagatesExisting verifies that existing request IDs are propagated
func TestWithRequestID_PropagatesExisting(t *testing.T) {
	t.Run("propagates existing X-Request-ID header", func(t *testing.T) {
		existingRequestID := "test-request-id-123"

		// Create test handler that captures the request
		var capturedRequestID string
		handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRequestID = GetRequestID(r)
			w.WriteHeader(http.StatusOK)
		}))

		// Create request with existing X-Request-ID header
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RequestIDHeader, existingRequestID)
		rec := httptest.NewRecorder()

		// Execute handler
		handler.ServeHTTP(rec, req)

		// Verify existing request ID was used
		if capturedRequestID != existingRequestID {
			t.Errorf("Expected request ID %s, got %s", existingRequestID, capturedRequestID)
		}

		// Verify response header contains request ID
		responseRequestID := rec.Header().Get(RequestIDHeader)
		if responseRequestID != existingRequestID {
			t.Errorf("Expected response header to match request ID, got %s != %s", responseRequestID, existingRequestID)
		}
	})
}

// TestWithRequestID_ResponseHeader verifies response header is always set
func TestWithRequestID_ResponseHeader(t *testing.T) {
	t.Run("sets X-Request-ID in response header", func(t *testing.T) {
		handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Verify response header is set
		responseRequestID := rec.Header().Get(RequestIDHeader)
		if responseRequestID == "" {
			t.Error("Expected X-Request-ID in response header, got empty string")
		}
	})
}

// TestGetRequestID_ReturnsEmpty verifies GetRequestID returns empty when no ID in context
func TestGetRequestID_ReturnsEmpty(t *testing.T) {
	t.Run("returns empty string when no request ID in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		requestID := GetRequestID(req)

		if requestID != "" {
			t.Errorf("Expected empty string, got %s", requestID)
		}
	})
}

// TestWithRequestID_ContextPropagation verifies request ID is stored in context
func TestWithRequestID_ContextPropagation(t *testing.T) {
	t.Run("stores request ID in context for downstream use", func(t *testing.T) {
		handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify context contains request ID
			requestID := GetRequestID(r)
			if requestID == "" {
				t.Error("Expected request ID in context, got empty string")
			}

			// Verify context value can be retrieved directly
			if ctxValue := r.Context().Value(RequestIDContextKey); ctxValue == nil {
				t.Error("Expected request ID in context via key, got nil")
			}

			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
	})
}

// TestWithRequestID_MultipleRequests verifies each request gets unique ID
func TestWithRequestID_MultipleRequests(t *testing.T) {
	t.Run("generates unique request IDs for different requests", func(t *testing.T) {
		var requestID1, requestID2 string

		handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request
		req1 := httptest.NewRequest("GET", "/test", nil)
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)
		requestID1 = rec1.Header().Get(RequestIDHeader)

		// Second request
		req2 := httptest.NewRequest("GET", "/test", nil)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		requestID2 = rec2.Header().Get(RequestIDHeader)

		// Verify IDs are different
		if requestID1 == requestID2 {
			t.Errorf("Expected unique request IDs, got same ID: %s", requestID1)
		}

		// Verify both are valid UUIDs
		if _, err := uuid.Parse(requestID1); err != nil {
			t.Errorf("First request ID is not a valid UUID: %v", err)
		}
		if _, err := uuid.Parse(requestID2); err != nil {
			t.Errorf("Second request ID is not a valid UUID: %v", err)
		}
	})
}

// BenchmarkWithRequestID benchmarks request ID middleware performance
func BenchmarkWithRequestID(b *testing.B) {
	handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetRequestID(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkWithRequestID_ExistingID benchmarks with existing request ID
func BenchmarkWithRequestID_ExistingID(b *testing.B) {
	handler := WithRequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetRequestID(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, "existing-request-id-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
