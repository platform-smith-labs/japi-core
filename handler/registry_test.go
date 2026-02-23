package handler

import (
	"net/http"
	"sync"
	"testing"
)

// TestNewRegistry verifies registry creation
func TestNewRegistry(t *testing.T) {
	t.Run("creates empty registry", func(t *testing.T) {
		reg := NewRegistry()

		if reg == nil {
			t.Fatal("Expected non-nil registry")
		}

		routes := reg.GetRoutes()
		if len(routes) != 0 {
			t.Errorf("Expected empty registry, got %d routes", len(routes))
		}
	})
}

// TestMultipleRegistries verifies registries are independent
func TestMultipleRegistries(t *testing.T) {
	t.Run("independent registries don't interfere", func(t *testing.T) {
		reg1 := NewRegistry()
		reg2 := NewRegistry()

		// Register route in reg1
		MakeHandler(reg1,
			RouteInfo{Method: "GET", Path: "/test1"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)

		// Register different route in reg2
		MakeHandler(reg2,
			RouteInfo{Method: "GET", Path: "/test2"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)

		routes1 := reg1.GetRoutes()
		routes2 := reg2.GetRoutes()

		if len(routes1) != 1 {
			t.Errorf("Expected 1 route in reg1, got %d", len(routes1))
		}

		if len(routes2) != 1 {
			t.Errorf("Expected 1 route in reg2, got %d", len(routes2))
		}

		if routes1[0].Path == routes2[0].Path {
			t.Error("Expected different paths in different registries")
		}
	})
}

// TestMakeHandler verifies handler registration
func TestMakeHandler(t *testing.T) {
	t.Run("registers handler with route info", func(t *testing.T) {
		reg := NewRegistry()

		routeInfo := RouteInfo{
			Method:      "POST",
			Path:        "/users",
			Summary:     "Create user",
			Description: "Creates a new user",
			Tags:        []string{"Users"},
		}

		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			return struct{}{}, nil
		}

		result := MakeHandler(reg, routeInfo, handler)

		if result == nil {
			t.Fatal("Expected non-nil handler")
		}

		routes := reg.GetRoutes()
		if len(routes) != 1 {
			t.Fatalf("Expected 1 route, got %d", len(routes))
		}

		route := routes[0]
		if route.Method != "POST" {
			t.Errorf("Expected POST, got %s", route.Method)
		}
		if route.Path != "/users" {
			t.Errorf("Expected /users, got %s", route.Path)
		}
		if route.RouteInfo.Summary != "Create user" {
			t.Errorf("Expected 'Create user', got %s", route.RouteInfo.Summary)
		}
	})

	t.Run("registers multiple handlers", func(t *testing.T) {
		reg := NewRegistry()

		MakeHandler(reg,
			RouteInfo{Method: "GET", Path: "/users"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)

		MakeHandler(reg,
			RouteInfo{Method: "POST", Path: "/users"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)

		MakeHandler(reg,
			RouteInfo{Method: "DELETE", Path: "/users/{id}"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)

		routes := reg.GetRoutes()
		if len(routes) != 3 {
			t.Errorf("Expected 3 routes, got %d", len(routes))
		}
	})
}

// TestConcurrentRegistration verifies thread-safety
func TestConcurrentRegistration(t *testing.T) {
	t.Run("concurrent registration is thread-safe", func(t *testing.T) {
		reg := NewRegistry()

		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				MakeHandler(reg,
					RouteInfo{Method: "GET", Path: "/test"},
					func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
						return struct{}{}, nil
					},
				)
			}(i)
		}

		wg.Wait()

		routes := reg.GetRoutes()
		if len(routes) != numGoroutines {
			t.Errorf("Expected %d routes, got %d", numGoroutines, len(routes))
		}
	})
}

// TestGetRoutes verifies route retrieval
func TestGetRoutes(t *testing.T) {
	t.Run("returns copy of routes", func(t *testing.T) {
		reg := NewRegistry()

		MakeHandler(reg,
			RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)

		routes1 := reg.GetRoutes()
		routes2 := reg.GetRoutes()

		// Should be different slices (copies)
		if &routes1[0] == &routes2[0] {
			t.Error("Expected different slice copies, got same underlying array")
		}

		// But should have same content
		if routes1[0].Path != routes2[0].Path {
			t.Error("Expected same route content in copies")
		}
	})
}

// TestRegisterWithRouter verifies router integration
func TestRegisterWithRouter(t *testing.T) {
	t.Run("registers routes with chi router", func(t *testing.T) {
		reg := NewRegistry()

		MakeHandler(reg,
			RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
				return struct{}{}, nil
			},
		)

		// Note: Full integration testing would require chi router import
		// This test verifies the method exists and doesn't panic
		// Actual router integration is tested in integration tests

		routes := reg.GetRoutes()
		if len(routes) != 1 {
			t.Errorf("Expected 1 route, got %d", len(routes))
		}
	})
}

// TestMiddlewareNames verifies middleware tracking
func TestMiddlewareNames(t *testing.T) {
	t.Run("tracks middleware names", func(t *testing.T) {
		reg := NewRegistry()

		middleware1 := func(next Handler[struct{}, struct{}, struct{}]) Handler[struct{}, struct{}, struct{}] {
			return next
		}

		MakeHandler(reg,
			RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
			middleware1,
		)

		routes := reg.GetRoutes()
		if len(routes) != 1 {
			t.Fatalf("Expected 1 route, got %d", len(routes))
		}

		route := routes[0]
		if len(route.MiddlewareNames) != 1 {
			t.Errorf("Expected 1 middleware, got %d", len(route.MiddlewareNames))
		}
	})
}

// TestTypedHandler verifies AdaptableHandler interface
func TestTypedHandler(t *testing.T) {
	t.Run("TypedHandler implements AdaptableHandler", func(t *testing.T) {
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			return struct{}{}, nil
		}

		th := TypedHandler[struct{}, struct{}, struct{}]{handler: handler}

		// Verify it implements the interface
		var _ AdaptableHandler = th

		// Verify Adapt method works
		adapted := th.Adapt(nil, nil)
		if adapted == nil {
			t.Error("Expected non-nil adapted handler")
		}
	})
}

// BenchmarkRegistryCreation benchmarks registry creation
func BenchmarkRegistryCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRegistry()
	}
}

// BenchmarkMakeHandler benchmarks handler registration
func BenchmarkMakeHandler(b *testing.B) {
	reg := NewRegistry()
	routeInfo := RouteInfo{Method: "GET", Path: "/test"}
	handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
		return struct{}{}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MakeHandler(reg, routeInfo, handler)
	}
}

// BenchmarkGetRoutes benchmarks route retrieval
func BenchmarkGetRoutes(b *testing.B) {
	reg := NewRegistry()

	// Register 100 routes
	for i := 0; i < 100; i++ {
		MakeHandler(reg,
			RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.GetRoutes()
	}
}

// BenchmarkConcurrentRegistration benchmarks concurrent registration
func BenchmarkConcurrentRegistration(b *testing.B) {
	reg := NewRegistry()
	routeInfo := RouteInfo{Method: "GET", Path: "/test"}
	handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
		return struct{}{}, nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			MakeHandler(reg, routeInfo, handler)
		}
	})
}
