# JAPI Core

A powerful, type-safe Go framework providing API and database layer abstractions for building modern web applications. Built with generics, automatic route registration, and comprehensive middleware support.

## Features

### Core API Framework
- **Type-Safe Handlers** - Generic handler system with compile-time type checking
- **Structured Error Handling** - Comprehensive error types with field-level validation
- **Automatic Route Registration** - Handlers self-register with metadata for documentation
- **Middleware Composition** - Functional middleware pattern with intuitive composition
- **Auto-Generated Swagger** - OpenAPI documentation generated from route metadata

### Database Abstraction
- **Connection Pooling** - PostgreSQL connection management with configurable pools
- **Transaction Wrapper** - Generic transaction handling with automatic rollback
- **Type-Safe Queries** - Automatic struct scanning for query results
- **Health Checks** - Built-in database health monitoring

### Authentication & Security
- **JWT Authentication** - Complete JWT token generation and validation
- **Auth Middleware** - Ready-to-use authentication middleware
- **CORS Support** - Configurable CORS middleware

### Validation & Parsing
- **Request Validation** - Automatic validation using struct tags
- **Multi-Format Support** - JSON, CSV, and multipart form parsing
- **Custom Validators** - Extensible validation system
- **Nullable Types** - Type-safe optional values

## Architecture

### Package Structure

```
japi-core/
├── core/           # Core types: errors, handlers, responses
├── handler/        # Generic handler framework and route registration
├── middleware/     # Standard HTTP and typed middleware
│   ├── http/       # Standard HTTP middleware (logging, content-type)
│   ├── typed/      # Generic typed middleware (auth, validation, parsing)
│   └── validation/ # Custom validator setup
├── db/             # Database connection and query abstractions
├── router/         # Chi router configuration
├── jwt/            # JWT token generation and validation
└── swagger/        # Auto-generated Swagger documentation
```

### Dependency Layers

**Layer 0 (Foundation)**
- `core/` - Core types and utilities
- `db/` - Database abstractions
- `jwt/` - JWT utilities

**Layer 1**
- `handler/` - Generic handler framework (depends on `core`)
- `router/` - Router setup (depends on `core`)

**Layer 2**
- `middleware/` - All middleware (depends on `core`, `handler`, `jwt`)
- `swagger/` - Documentation generation (depends on `handler`)

## Installation

```bash
go get github.com/platform-smith-labs/japi-core
```

## Quick Start

### 1. Setup Database and Router

```go
package main

import (
    "log"
    "log/slog"
    "net/http"
    "os"

    "github.com/platform-smith-labs/japi-core/db"
    "github.com/platform-smith-labs/japi-core/router"
    "github.com/platform-smith-labs/japi-core/handler"
    "github.com/platform-smith-labs/japi-core/swagger"
)

func main() {
    // Setup logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Connect to database
    // Connection pool settings use sensible defaults if omitted (25/25/5min/5min)
    dbConn, err := db.Connect(db.Config{
        Host:         "localhost",
        Port:         5432,
        User:         "postgres",
        Password:     "password",
        Database:     "myapp",
        SSLMode:      "disable",
        MaxOpenConns: 25,                  // Max open connections (default: 25)
        MaxIdleConns: 25,                  // Max idle connections (default: 25)
        MaxLifetime:  5 * time.Minute,     // Connection max lifetime (default: 5min)
        MaxIdleTime:  5 * time.Minute,     // Connection max idle time (default: 5min)
    })
    if err != nil {
        log.Fatal(err)
    }
    defer dbConn.Close()

    // Create router with CORS configuration
    // IMPORTANT: Default router denies all cross-origin requests (secure default)
    // For production APIs with web frontends, configure allowed origins:
    r := router.NewChiRouterWithCORS([]string{
        "https://yourdomain.com",
        "https://app.yourdomain.com",
    })

    // For local development only:
    // r := router.NewChiRouterWithCORS([]string{"http://localhost:3000"})

    // Or use default router and configure CORS manually later
    // r := router.NewChiRouter() // Denies all origins by default

    // Create a registry for your routes
    registry := handler.NewRegistry()

    // Import your handlers (they will register with the registry)
    // import _ "your-app/handlers" // if using package-level registration

    // Register all routes with the router
    registry.RegisterWithRouter(r, dbConn, logger)

    // Setup Swagger UI
    swagger.SwaggerInfo.Title = "My API"
    swagger.SwaggerInfo.Description = "API documentation"
    swagger.SwaggerInfo.Version = "1.0.0"
    swagger.SetupSwaggerUI(r, registry)

    // Start server
    logger.Info("Server starting on :8080")
    http.ListenAndServe(":8080", r)
}
```

### 2. Create Your First Handler

```go
package handlers

import (
    "database/sql"
    "net/http"

    "github.com/platform-smith-labs/japi-core/core"
    "github.com/platform-smith-labs/japi-core/handler"
    "github.com/platform-smith-labs/japi-core/middleware/typed"
)

// Request/Response types
type CreateUserParams struct {
    // No URL parameters in this example
}

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Handler implementation
func CreateUser(ctx *handler.HandlerContext[CreateUserParams, CreateUserRequest]) (*CreateUserResponse, *core.APIError) {
    // Access validated request body
    req := ctx.Body.Value()

    // Insert into database
    var userID int
    err := ctx.DB.QueryRowContext(ctx.Context,
        "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
        req.Name, req.Email,
    ).Scan(&userID)

    if err != nil {
        return nil, core.NewAPIError(
            http.StatusInternalServerError,
            "Failed to create user",
            err.Error(),
        )
    }

    // Return response
    return &CreateUserResponse{
        ID:    userID,
        Name:  req.Name,
        Email: req.Email,
    }, nil
}

// Package-level registry (one per package/server)
var Server = handler.NewRegistry()

// Register handler with automatic route registration
var CreateUserHandler = handler.MakeHandler(
    Server, // Pass the registry
    handler.RouteInfo{
        Method:      "POST",
        Path:        "/users",
        Summary:     "Create a new user",
        Description: "Creates a new user with the provided name and email",
        Tags:        []string{"Users"},
    },
    CreateUser,
    typed.ParseBody[CreateUserParams, CreateUserRequest, *CreateUserResponse](),
    typed.ResponseJSON[CreateUserParams, CreateUserRequest, *CreateUserResponse](),
)
```

## Usage Examples

### Working with URL Parameters

```go
type GetUserParams struct {
    UserID string `param:"id" validate:"required,uuid"`
}

type GetUserRequest struct {
    // No request body
}

type GetUserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func GetUser(ctx *handler.HandlerContext[GetUserParams, GetUserRequest]) (*GetUserResponse, *core.APIError) {
    params := ctx.Params.Value()

    var user GetUserResponse
    err := ctx.DB.QueryRowContext(ctx.Context,
        "SELECT id, name, email FROM users WHERE id = $1",
        params.UserID,
    ).Scan(&user.ID, &user.Name, &user.Email)

    if err == sql.ErrNoRows {
        return nil, core.ErrNotFound("User not found")
    }
    if err != nil {
        return nil, core.ErrInternal(err.Error())
    }

    return &user, nil
}

var GetUserHandler = handler.MakeHandler(
    handler.RouteInfo{
        Method: "GET",
        Path:   "/users/{id}",
        Summary: "Get user by ID",
        Tags:   []string{"Users"},
    },
    GetUser,
    typed.ParseParams[GetUserParams, GetUserRequest, *GetUserResponse](),
    typed.ResponseJSON[GetUserParams, GetUserRequest, *GetUserResponse](),
)
```

### Using Authentication Middleware

```go
import "github.com/platform-smith-labs/japi-core/middleware/typed"

type ProtectedParams struct{}
type ProtectedRequest struct{}
type ProtectedResponse struct {
    Message string `json:"message"`
    UserID  string `json:"user_id"`
}

func ProtectedHandler(ctx *handler.HandlerContext[ProtectedParams, ProtectedRequest]) (*ProtectedResponse, *core.APIError) {
    // UserUUID is set by RequireAuth middleware
    return &ProtectedResponse{
        Message: "This is a protected endpoint",
        UserID:  ctx.UserUUID.String(),
    }, nil
}

var ProtectedHandlerRegistration = handler.MakeHandler(
    handler.RouteInfo{
        Method:  "GET",
        Path:    "/protected",
        Summary: "Protected endpoint requiring authentication",
        Tags:    []string{"Auth"},
    },
    ProtectedHandler,
    // Add authentication middleware
    typed.RequireAuth[ProtectedParams, ProtectedRequest, *ProtectedResponse](
        "your-jwt-secret",
        true, // validate user exists in database
    ),
    typed.ResponseJSON[ProtectedParams, ProtectedRequest, *ProtectedResponse](),
)
```

### Database Queries with Type Safety

```go
import "github.com/platform-smith-labs/japi-core/db"

type User struct {
    ID    string `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

// Query single row
func getUserByID(ctx context.Context, database *sql.DB, userID string) (*User, error) {
    return db.QueryOne[User](
        ctx,      // Context for cancellation/timeout
        database,
        "SELECT id, name, email FROM users WHERE id = $1",
        userID,
    )
}

// Query multiple rows
func getAllUsers(ctx context.Context, database *sql.DB) ([]User, error) {
    return db.QueryMany[User](
        ctx,      // Context for cancellation/timeout
        database,
        "SELECT id, name, email FROM users ORDER BY name",
    )
}

// Transaction wrapper
func transferCredits(ctx context.Context, database *sql.DB, fromUserID, toUserID string, amount int) error {
    return db.WithTx(ctx, database, func(txCtx context.Context, tx *sql.Tx) error {
        // Deduct from sender
        _, err := db.Exec(txCtx, tx,
            "UPDATE users SET credits = credits - $1 WHERE id = $2",
            amount, fromUserID,
        )
        if err != nil {
            return err
        }

        // Add to receiver
        _, err = db.Exec(txCtx, tx,
            "UPDATE users SET credits = credits + $1 WHERE id = $2",
            amount, toUserID,
        )
        return err
    })
}
```

### Context Propagation and Cancellation

japi-core v2.0.0 introduces comprehensive context propagation throughout the framework, enabling production-ready request cancellation and timeout handling.

#### What is Context Propagation?

Context propagation means that the HTTP request context (`r.Context()`) is automatically threaded through your handlers and into database queries. This enables:

- **✅ Automatic Cancellation** - Database queries stop when clients disconnect
- **✅ Timeout Support** - Queries respect request-level timeouts
- **✅ Resource Efficiency** - No wasted database connections or CPU cycles
- **✅ Distributed Tracing** - Context can carry trace IDs across service boundaries
- **✅ Production Reliability** - Handles edge cases like slow clients and network issues

#### How It Works

```go
func GetUser(ctx handler.HandlerContext[GetUserParams, GetUserRequest]) (*GetUserResponse, error) {
    params := ctx.Params.Value()

    // ctx.Context is automatically set from r.Context() by the adapter
    // When the client disconnects, ctx.Context is cancelled
    // The database query will be interrupted
    user, err := db.QueryOne[User](
        ctx.Context,  // Propagate request context to database
        ctx.DB,
        "SELECT * FROM users WHERE id = $1",
        params.UserID,
    )

    if err != nil {
        // Check for context-specific errors
        if errors.Is(err, context.Canceled) {
            // Client disconnected - log and return
            ctx.Logger.Info("Request cancelled by client")
            return nil, core.NewAPIError(499, "Client closed request")
        }
        if errors.Is(err, context.DeadlineExceeded) {
            // Request timeout
            ctx.Logger.Error("Request timeout")
            return nil, core.NewAPIError(504, "Request timeout")
        }
        return nil, core.ErrInternal(err.Error())
    }

    return &GetUserResponse{User: user}, nil
}
```

#### Example: Handling Timeouts

```go
// In main.go, configure timeout middleware
r := router.NewChiRouter()
r.Use(middleware.Timeout(30 * time.Second)) // 30-second timeout

// In handler
func ExportLargeReport(ctx handler.HandlerContext[ExportParams, ExportRequest]) (*ExportResponse, error) {
    // This query will be cancelled after 30 seconds
    results, err := db.QueryMany[ReportRow](
        ctx.Context,  // Timeout automatically propagated
        ctx.DB,
        "SELECT * FROM large_table WHERE created_at > $1",
        startDate,
    )

    if errors.Is(err, context.DeadlineExceeded) {
        return nil, core.NewAPIError(504, "Report generation timed out")
    }

    // Process results...
    return &ExportResponse{Data: results}, nil
}
```

#### Example: Client Disconnect Detection

```go
func ProcessLongRunningTask(ctx handler.HandlerContext[TaskParams, TaskRequest]) (*TaskResponse, error) {
    // Start processing
    for i := 0; i < 1000000; i++ {
        // Check if client disconnected every 1000 iterations
        if i%1000 == 0 {
            select {
            case <-ctx.Context.Done():
                // Client disconnected, stop processing
                ctx.Logger.Info("Client disconnected, stopping task",
                    "progress", i,
                    "error", ctx.Context.Err(),
                )
                return nil, core.NewAPIError(499, "Client closed request")
            default:
                // Continue processing
            }
        }

        // Do work...
    }

    return &TaskResponse{Processed: 1000000}, nil
}
```

#### Transaction Rollback on Cancellation

Transactions automatically rollback when the context is cancelled:

```go
func TransferFunds(ctx handler.HandlerContext[TransferParams, TransferRequest]) (*TransferResponse, error) {
    req := ctx.Body.Value()

    err := db.WithTx(ctx.Context, ctx.DB, func(txCtx context.Context, tx *sql.Tx) error {
        // If client disconnects during this transaction, it will be rolled back
        _, err := db.Exec(txCtx, tx,
            "UPDATE accounts SET balance = balance - $1 WHERE id = $2",
            req.Amount, req.FromAccount,
        )
        if err != nil {
            return err
        }

        // Simulating slow operation - if context cancelled, transaction rolls back
        time.Sleep(2 * time.Second)

        _, err = db.Exec(txCtx, tx,
            "UPDATE accounts SET balance = balance + $1 WHERE id = $2",
            req.Amount, req.ToAccount,
        )
        return err
    })

    if errors.Is(err, context.Canceled) {
        ctx.Logger.Info("Transaction cancelled - funds transfer rolled back")
        return nil, core.NewAPIError(499, "Transfer cancelled")
    }

    return &TransferResponse{Success: true}, nil
}
```

#### Migration from v1.x

If you're migrating from japi-core v1.x (without context support), see [MIGRATION.md](MIGRATION.md) for a comprehensive guide with step-by-step instructions and before/after examples.

**Key Changes:**
- All database functions now require `context.Context` as the first parameter
- `HandlerContext` has a new `Context` field (automatically set by adapter)
- Transaction callbacks receive both `context.Context` and `*sql.Tx`

```go
// v1.x
users, err := db.QueryMany[User](ctx.DB, query, args...)

// v2.0.0
users, err := db.QueryMany[User](ctx.Context, ctx.DB, query, args...)
```

### Multi-Server Applications

japi-core v3.0.0 supports running multiple servers with independent route sets in the same application. This is useful for:
- Separating public API and admin interfaces
- Running services on different ports with different routes
- Microservices architectures
- Different authentication requirements per server

#### Basic Multi-Server Setup

```go
// handlers1/handlers.go (API Server)
package handlers1

import "github.com/platform-smith-labs/japi-core/handler"

// One registry per server/package
var Server = handler.NewRegistry()

var GetUser = handler.MakeHandler(
    Server,
    handler.RouteInfo{Method: "GET", Path: "/users/{id}"},
    getUserHandler,
    typed.ParseParams[...](),
    typed.ResponseJSON[...](),
)

var CreateUser = handler.MakeHandler(
    Server,
    handler.RouteInfo{Method: "POST", Path: "/users"},
    createUserHandler,
    typed.ParseBody[...](),
    typed.ResponseJSON[...](),
)
```

```go
// handlers2/handlers.go (Admin Server)
package handlers2

import "github.com/platform-smith-labs/japi-core/handler"

// Separate registry for admin server
var Server = handler.NewRegistry()

var AdminDashboard = handler.MakeHandler(
    Server,
    handler.RouteInfo{Method: "GET", Path: "/dashboard"},
    dashboardHandler,
    typed.RequireAuth[...]("secret", true),
    typed.ResponseJSON[...](),
)
```

```go
// main.go
package main

import (
    "your-app/handlers1"
    "your-app/handlers2"
)

func main() {
    db, _ := db.Connect(config)
    logger := slog.Default()

    // API Server on :8080
    apiRouter := router.NewChiRouter()
    handlers1.Server.RegisterWithRouter(apiRouter, db, logger)
    swagger.SetupSwaggerUI(apiRouter, handlers1.Server)
    go http.ListenAndServe(":8080", apiRouter)

    // Admin Server on :8081
    adminRouter := router.NewChiRouter()
    handlers2.Server.RegisterWithRouter(adminRouter, db, logger)
    swagger.SetupSwaggerUI(adminRouter, handlers2.Server)
    http.ListenAndServe(":8081", adminRouter)
}
```

#### Benefits of Multi-Server Architecture

✅ **Isolated route sets** - Each server has completely independent routes
✅ **Different middleware** - Apply different auth/cors/logging per server
✅ **Independent scaling** - Scale API and admin separately
✅ **Clear separation** - Organize code by server responsibility
✅ **Test isolation** - Test each server independently

### Connection Pool Configuration

The database connection pool is critical for application performance and stability. japi-core uses **production-safe defaults** that work well for most applications.

#### Default Values

If you omit connection pool settings, these defaults are applied:

- **MaxOpenConns: 25** - Maximum number of open connections to the database
- **MaxIdleConns: 25** - Maximum number of idle connections in the pool
- **MaxLifetime: 5 minutes** - Maximum time a connection can be reused
- **MaxIdleTime: 5 minutes** - Maximum time a connection can be idle before being closed

#### Why These Defaults?

**MaxOpenConns = 25**: Prevents unlimited connections that could overwhelm your database or cause out-of-memory errors. 25 is a reasonable starting point for small-to-medium applications.

**MaxIdleConns = 25**: Keeping all connections idle (equal to MaxOpenConns) minimizes connection churn and reduces latency for high-throughput services. Connections are kept warm and ready to use.

**MaxLifetime = 5 minutes**: Prevents stale connections by recycling them periodically. Many load balancers, proxies, and databases close idle connections after 5-10 minutes, so staying below that threshold avoids connection failures.

**MaxIdleTime = 5 minutes**: Allows the pool to scale down gracefully during low-traffic periods by closing truly idle connections. This reclaims resources without hurting performance.

#### Configuration Examples

**Minimal Configuration (Uses Defaults)**

```go
dbConn, err := db.Connect(db.Config{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "password",
    Database: "myapp",
    SSLMode:  "disable",
    // MaxOpenConns, MaxIdleConns, MaxLifetime, MaxIdleTime use defaults
})
```

**Small Application (Low Concurrency)**

```go
dbConn, err := db.Connect(db.Config{
    Host:         "localhost",
    Port:         5432,
    User:         "postgres",
    Password:     "password",
    Database:     "myapp",
    SSLMode:      "disable",
    MaxOpenConns: 10,
    MaxIdleConns: 5,
    MaxLifetime:  5 * time.Minute,
    MaxIdleTime:  3 * time.Minute,
})
```

**High-Throughput Application**

```go
dbConn, err := db.Connect(db.Config{
    Host:         "localhost",
    Port:         5432,
    User:         "postgres",
    Password:     "password",
    Database:     "myapp",
    SSLMode:      "require",
    MaxOpenConns: 100,
    MaxIdleConns: 100,
    MaxLifetime:  5 * time.Minute,
    MaxIdleTime:  5 * time.Minute,
})
```

#### Tuning Guidelines

**When to Increase MaxOpenConns:**
- Application handles high concurrent request volume
- Database server has sufficient resources
- Load testing shows connection pool exhaustion
- **Rule of thumb**: `MaxOpenConns = (Available DB Connections) / (Number of App Instances)`

**When to Decrease MaxOpenConns:**
- Small application with low concurrency
- Shared database with limited connection slots
- Want to reduce database load

**When to Lower MaxIdleConns:**
- Low-traffic applications where connection churn is acceptable
- Want to reclaim idle connection resources faster
- Database enforces connection limits
- **Common pattern**: Set to 20-30% of MaxOpenConns

**When to Keep MaxIdleConns = MaxOpenConns:**
- High-throughput services where latency matters
- Consistent traffic patterns
- Want to avoid connection establishment overhead

#### Validation

The library validates your configuration:

```go
// This will return an error
dbConn, err := db.Connect(db.Config{
    MaxOpenConns: 10,
    MaxIdleConns: 25, // ERROR: Cannot exceed MaxOpenConns
    // ...
})
// err: "MaxIdleConns (25) cannot exceed MaxOpenConns (10)"
```

#### Monitoring Connection Pool Health

```go
import "database/sql"

// Get connection pool statistics
stats := dbConn.Stats()
log.Printf("Open connections: %d", stats.OpenConnections)
log.Printf("In use: %d", stats.InUse)
log.Printf("Idle: %d", stats.Idle)
log.Printf("Wait count: %d", stats.WaitCount)
log.Printf("Wait duration: %v", stats.WaitDuration)
log.Printf("Max idle closed: %d", stats.MaxIdleClosed)
log.Printf("Max lifetime closed: %d", stats.MaxLifetimeClosed)
```

**Warning Signs:**
- High `WaitCount` → Increase MaxOpenConns
- High `WaitDuration` → Pool is saturated, increase MaxOpenConns
- High `MaxIdleClosed` → MaxIdleConns might be too low for your traffic pattern
- `OpenConnections` always at `MaxOpenConns` → Consider increasing limit

### Working with Nullable Types

Nullable types provide explicit optionality for request-scoped data. Middleware populates Nullable values, and handlers can safely assume they're present.

```go
import "github.com/platform-smith-labs/japi-core/handler"

type UpdateUserRequest struct {
    Name  handler.Nullable[string] `json:"name"`
    Email handler.Nullable[string] `json:"email"`
}

func UpdateUser(ctx *handler.HandlerContext[UpdateUserParams, UpdateUserRequest]) (*UserResponse, *core.APIError) {
    // Safe: ParseBody middleware guarantees ctx.Body is populated
    req := ctx.Body.Value()

    // Check if name was provided
    if req.Name.HasValue() {
        // Use the value safely
        name := req.Name.Value()
        // ... update name
    }

    // Get value or default
    email := req.Email.ValueOrDefault()

    return &UserResponse{}, nil
}
```

**Important: Value() Panic Behavior**

`Nullable.Value()` implements fail-fast behavior: it panics (with a recoverable panic) if called on an empty Nullable. This is intentional:

```go
// ✅ SAFE - Middleware guarantees value is set
func CreateUser(ctx HandlerContext) (*Response, error) {
    body := ctx.Body.Value() // Never panics if ParseBody middleware applied
    // Use body...
}

// ❌ UNSAFE - Calling Value() without checking
func HandleOptional(ctx HandlerContext) (*Response, error) {
    // This will panic if UserUUID not set by RequireAuth middleware
    userID := ctx.UserUUID.Value()
    // ...
}

// ✅ SAFE - Check before accessing optional values
func HandleOptional(ctx HandlerContext) (*Response, error) {
    if ctx.UserUUID.HasValue() {
        userID := ctx.UserUUID.Value() // Safe
        // User is authenticated
    } else {
        // User not authenticated, handle accordingly
    }
}

// ✅ SAFE - Use TryValue for optional fields
func HandleOptional(ctx HandlerContext) (*Response, error) {
    if userID, ok := ctx.UserUUID.TryValue(); ok {
        // User is authenticated
    }
}
```

**Panic Recovery (Advanced)**

If needed, you can recover from Nullable panics in middleware:

```go
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if r := recover(); r != nil {
                // Log and return error response
                log.Printf("Panic recovered: %v", r)
                http.Error(w, "Internal Server Error", 500)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### CSV File Upload Handler

```go
import "github.com/platform-smith-labs/japi-core/middleware/typed"

type ImportUsersParams struct{}

type UserCSVRow struct {
    Name  string `csv:"name"`
    Email string `csv:"email"`
}

type ImportUsersRequest []UserCSVRow

type ImportUsersResponse struct {
    Imported int `json:"imported"`
}

func ImportUsers(ctx *handler.HandlerContext[ImportUsersParams, ImportUsersRequest]) (*ImportUsersResponse, *core.APIError) {
    users := ctx.Body.Value()

    for _, user := range users {
        // Insert user into database
        _, err := ctx.DB.ExecContext(ctx.Context,
            "INSERT INTO users (name, email) VALUES ($1, $2)",
            user.Name, user.Email,
        )
        if err != nil {
            return nil, core.ErrInternal(err.Error())
        }
    }

    return &ImportUsersResponse{Imported: len(users)}, nil
}

var ImportUsersHandler = handler.MakeHandler(
    handler.RouteInfo{
        Method:  "POST",
        Path:    "/users/import",
        Summary: "Import users from CSV",
        Tags:    []string{"Users"},
    },
    ImportUsers,
    // Use CSV parsing middleware instead of JSON
    typed.ParseCSV[ImportUsersParams, ImportUsersRequest, *ImportUsersResponse](),
    typed.ResponseJSON[ImportUsersParams, ImportUsersRequest, *ImportUsersResponse](),
)
```

### Custom Error Responses

```go
func CreateOrder(ctx *handler.HandlerContext[CreateOrderParams, CreateOrderRequest]) (*CreateOrderResponse, *core.APIError) {
    req := ctx.Body.Value()

    // Validation error with field-level details
    if req.Quantity <= 0 {
        return nil, core.NewValidationError("Invalid order").
            AddField("quantity", "must be greater than 0")
    }

    // Business logic error
    if req.Total < 100 {
        return nil, core.NewAPIError(
            http.StatusBadRequest,
            "Order total too low",
            "Minimum order total is $100",
        )
    }

    // Database constraint error handling
    err := ctx.DB.QueryRowContext(ctx.Context, "INSERT INTO orders ...").Scan(&orderID)
    if core.IsUniqueConstraintError(err) {
        return nil, core.NewAPIError(
            http.StatusConflict,
            "Duplicate order",
            "An order with this reference already exists",
        )
    }

    return &CreateOrderResponse{}, nil
}
```

### Swagger Documentation

The framework automatically generates OpenAPI/Swagger documentation from your handler metadata:

```go
// In main.go
swagger.SwaggerInfo.Title = "My API"
swagger.SwaggerInfo.Description = "Comprehensive API documentation"
swagger.SwaggerInfo.Version = "1.0.0"
swagger.SwaggerInfo.Host = "api.example.com"
swagger.SwaggerInfo.BasePath = "/v1"
swagger.SwaggerInfo.Schemes = []string{"https"}

swagger.SetupSwaggerUI(r)
```

Access Swagger UI at: `http://localhost:8080/swagger/index.html`

### Custom Validators

The library provides documentation and examples in `middleware/validation/setup.go` showing how to implement custom validators. Here's how to create your own:

```go
import (
    "database/sql"
    "github.com/go-playground/validator/v10"
)

func main() {
    // ... database setup ...

    validate := validator.New()

    // Register custom database-backed validators
    // These are specific to YOUR database schema
    validate.RegisterValidation("unique_email", uniqueEmailValidator(dbConn))
    validate.RegisterValidation("user_exists", userExistsValidator(dbConn))

    // Register custom business rule validators
    validate.RegisterValidation("valid_status", func(fl validator.FieldLevel) bool {
        status := fl.Field().String()
        validStatuses := []string{"active", "inactive", "pending"}
        for _, v := range validStatuses {
            if status == v {
                return true
            }
        }
        return false
    })

    // ... rest of setup ...
}

// Example: Unique email validator for YOUR database
func uniqueEmailValidator(db *sql.DB) validator.Func {
    return func(fl validator.FieldLevel) bool {
        email := fl.Field().String()
        if email == "" {
            return true // Let 'required' tag handle empty values
        }

        var count int
        // REPLACE 'users' with your actual table name
        // Note: Using context.Background() since validators don't have request context
        // For production with cancellation support, consider async validation
        err := db.QueryRowContext(context.Background(),
            "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
        if err != nil {
            return false
        }
        return count == 0
    }
}

// Example: User exists validator for YOUR database
func userExistsValidator(db *sql.DB) validator.Func {
    return func(fl validator.FieldLevel) bool {
        userID := fl.Field().String()
        if userID == "" {
            return false
        }

        var exists bool
        // REPLACE 'users' with your actual table name
        // Note: Using context.Background() since validators don't have request context
        err := db.QueryRowContext(context.Background(),
            "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
        if err != nil {
            return false
        }
        return exists
    }
}
```

**Note:** The library does NOT provide pre-built database validators because they would contain hardcoded table names that won't work for your project. See `middleware/validation/setup.go` for more examples and best practices.

## API Reference

### Core Package

#### Error Functions
- `core.NewAPIError(code, message, detail)` - Create API error
- `core.NewValidationError(message)` - Create validation error
- `core.ErrBadRequest(detail)` - 400 Bad Request
- `core.ErrUnauthorized(detail)` - 401 Unauthorized
- `core.ErrForbidden(detail)` - 403 Forbidden
- `core.ErrNotFound(detail)` - 404 Not Found
- `core.ErrInternal(detail)` - 500 Internal Server Error

#### Response Functions
- `core.Success[T](w, data)` - 200 OK response
- `core.Created[T](w, data)` - 201 Created response
- `core.NoContent(w)` - 204 No Content response
- `core.List[T](w, data, count)` - Paginated list response
- `core.Error(w, logger, err)` - Error response

### Handler Package

#### Types
- `HandlerContext[ParamTypeT, BodyTypeT]` - Handler context with DB, Logger, Params, Body, UserUUID, etc.
- `Handler[ParamTypeT, BodyTypeT, ResponseBodyT]` - Generic handler function type
- `Nullable[T]` - Optional value wrapper

#### Functions
- `handler.MakeHandler(routeInfo, handler, ...middleware)` - Create and register handler
- `handler.RegisterCollectedRoutes(router, db, logger)` - Register all routes with router

#### Nullable Methods
- `nullable.HasValue()` - Check if value exists
- `nullable.Value()` - Get value (panics if absent - recoverable panic)
- `nullable.TryValue()` - Safe value extraction with boolean (never panics)
- `nullable.ValueOrDefault()` - Get value or zero value (never panics)
- `nullable.ValueOr(default)` - Get value or provided default (never panics)

### Database Package

#### Functions
- `db.Connect(config)` - Establish database connection
- `db.HealthCheck(db)` - Database health check
- `db.QueryOne[T](ctx, querier, query, args...)` - Query single row (with cancellation/timeout)
- `db.QueryMany[T](ctx, querier, query, args...)` - Query multiple rows (with cancellation/timeout)
- `db.Exec(ctx, querier, query, args...)` - Execute query (with cancellation/timeout)
- `db.WithTx[T](ctx, db, fn)` - Transaction wrapper (with cancellation/timeout)

**Note:** All database operations now require `context.Context` as the first parameter for proper cancellation and timeout support.

### Middleware Package

#### Typed Middleware
- `typed.ParseParams[...]()` - Parse URL/query parameters
- `typed.ParseBody[...]()` - Parse JSON request body
- `typed.ParseHeaders[...]()` - Capture HTTP headers
- `typed.ParseCSV[...]()` - Parse CSV file upload
- `typed.ParseJSON[...]()` - Parse JSON file upload
- `typed.ResponseJSON[...]()` - Write JSON response
- `typed.ResponseJSONFile[...](filename)` - Write downloadable JSON file
- `typed.RequireAuth[...](jwtSecret, validateUser)` - JWT authentication
- `typed.WithRequestID` - Enrich context with request ID for tracing (types inferred)
- `typed.WithLogging` - Structured logging with timing (types inferred)

#### HTTP Middleware
- `http.WithRequestID()` - Generate/propagate request IDs for correlation
- `http.WithLogging(logger)` - Standard HTTP logging
- `http.WithContentType(contentType)` - Set response Content-Type

### JWT Package

#### Functions
- `jwt.GenerateToken(claims, secret, expiration)` - Generate JWT token
- `jwt.ValidateToken(tokenString, secret)` - Validate and parse JWT
- `jwt.ExtractClaims(tokenString)` - Extract claims without validation

### Router Package

#### Functions
- `router.NewChiRouter()` - Create Chi router with standard middleware (CORS, logging, etc.)

## Migrating Existing Projects

To migrate an existing project to use japi-core:

1. **Install the library**
   ```bash
   go get github.com/platform-smith-labs/japi-core
   ```

2. **Update imports**
   Replace your custom API framework imports with japi-core packages:
   ```go
   import (
       "github.com/platform-smith-labs/japi-core/core"
       "github.com/platform-smith-labs/japi-core/handler"
       "github.com/platform-smith-labs/japi-core/middleware/typed"
       "github.com/platform-smith-labs/japi-core/db"
   )
   ```

3. **Refactor handlers**
   Convert existing handlers to use the generic handler pattern with `HandlerContext`.

4. **Setup route registration**
   Replace manual route registration with `handler.MakeHandler()` and `handler.RegisterCollectedRoutes()`.

5. **Update database queries**
   Use `db.QueryOne()`, `db.QueryMany()`, and `db.WithTx()` for type-safe queries.

## Security

### CORS Configuration

**IMPORTANT**: By default, japi-core uses **secure CORS defaults** that deny all cross-origin requests. This prevents unauthorized websites from accessing your API.

#### Why Secure Defaults Matter

Insecure CORS configuration (`AllowedOrigins: ["*"]`) is a common security vulnerability that allows:
- **Data theft** - Any malicious website can read your API responses
- **Unauthorized actions** - Attackers can make requests on behalf of users
- **Session hijacking** - Credentials and tokens can be exposed

#### Configuring CORS Properly

**Option 1: Using NewChiRouterWithCORS (Recommended)**

```go
// Production: Explicitly allow your domains
r := router.NewChiRouterWithCORS([]string{
    "https://yourdomain.com",
    "https://app.yourdomain.com",
    "https://admin.yourdomain.com",
})
```

**Option 2: Manual Configuration**

```go
r := router.NewChiRouter() // Denies all by default

// Add CORS middleware with your allowed origins
r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"https://yourdomain.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
    AllowCredentials: false,
    MaxAge:           300,
}))
```

**For Local Development:**

```go
// Only for development! Never use in production
r := router.NewChiRouterWithCORS([]string{
    "http://localhost:3000",
    "http://localhost:5173", // Vite dev server
})
```

**What NOT to do:**

```go
// NEVER DO THIS IN PRODUCTION!
r := router.NewChiRouterWithCORS([]string{"*"}) // Allows ANY origin - security risk!
```

#### Environment-Based Configuration

```go
func getAllowedOrigins() []string {
    if os.Getenv("ENV") == "production" {
        return []string{
            "https://yourdomain.com",
            "https://app.yourdomain.com",
        }
    }
    // Development
    return []string{
        "http://localhost:3000",
        "http://localhost:5173",
    }
}

r := router.NewChiRouterWithCORS(getAllowedOrigins())
```

### Other Security Best Practices

1. **Always use HTTPS in production** - Protect data in transit
2. **Validate JWT secrets** - Use strong, randomly generated secrets (min 32 characters)
3. **Enable rate limiting** - Prevent abuse and DoS attacks (add custom middleware)
4. **Validate all inputs** - Use struct validation tags extensively
5. **Use prepared statements** - Prevent SQL injection (already done by `db` package)
6. **Set security headers** - Add CSP, HSTS, X-Frame-Options headers
7. **Sanitize error messages** - Don't expose internal details in production

## Observability

### Request ID Middleware

Request IDs are unique identifiers assigned to each HTTP request for correlation and tracing across distributed systems. japi-core provides built-in middleware for generating and propagating request IDs.

#### Why Request IDs Matter

- **Distributed Tracing** - Track requests across microservices
- **Log Correlation** - Group all logs from the same request
- **Debugging** - Identify problematic requests in production
- **Customer Support** - Reference specific requests when troubleshooting
- **Performance Analysis** - Track request latency across services

#### Usage

**Step 1: Apply HTTP Middleware**

Add `http.WithRequestID()` to your router to generate/propagate request IDs:

```go
import (
    "github.com/platform-smith-labs/japi-core/router"
    httpMiddleware "github.com/platform-smith-labs/japi-core/middleware/http"
)

func main() {
    r := router.NewChiRouter()

    // Apply request ID middleware early in the chain
    r.Use(httpMiddleware.WithRequestID())

    // ... register routes ...
}
```

**Step 2: Apply Typed Middleware**

Enrich your `HandlerContext` with the request ID using `typed.WithRequestID()`:

```go
import (
    "github.com/platform-smith-labs/japi-core/handler"
    "github.com/platform-smith-labs/japi-core/middleware/typed"
)

var CreateUser = handler.MakeHandler(
    Server,
    handler.RouteInfo{Method: "POST", Path: "/users"},
    createUserHandler,
    typed.WithRequestID,  // Type parameters inferred from createUserHandler!
    typed.WithLogging,
)
```

**Step 3: Access Request ID in Handlers**

```go
func createUserHandler(
    ctx handler.HandlerContext[CreateUserParams, CreateUserBody],
    w http.ResponseWriter,
    r *http.Request,
) (UserResponse, error) {
    // Access request ID from context
    if ctx.RequestID.HasValue() {
        requestID := ctx.RequestID.Value()
        ctx.Logger.Info("Creating user", "request_id", requestID)
    }

    // Your business logic...
}
```

#### Request ID Propagation

**Incoming Requests**

The `http.WithRequestID()` middleware:
1. Reads `X-Request-ID` header from incoming requests
2. Generates a new UUID if no header is present
3. Stores the request ID in the request context
4. Adds `X-Request-ID` to response headers

**Outgoing Requests**

When calling other services, propagate the request ID:

```go
func callExternalService(ctx handler.HandlerContext[P, B], url string) error {
    req, _ := http.NewRequest("GET", url, nil)

    // Propagate request ID to downstream service
    if ctx.RequestID.HasValue() {
        req.Header.Set("X-Request-ID", ctx.RequestID.Value())
    }

    // Make request...
}
```

#### Structured Logging with Request IDs

The `typed.WithRequestID()` middleware automatically enriches your logger with the request ID:

```go
// Logger automatically includes request_id in all log statements
ctx.Logger.Info("User created successfully", "user_id", user.ID)
// Output: {"level":"INFO","msg":"User created successfully","user_id":"123","request_id":"550e8400-e29b-41d4-a716-446655440000"}
```

#### Type Inference

Go automatically infers type parameters for `WithRequestID` and `WithLogging` middleware when used in `MakeHandler`:

```go
// Type parameters automatically inferred from createUserHandler signature ✨
var CreateUser = handler.MakeHandler(
    Server,
    handler.RouteInfo{Method: "POST", Path: "/users"},
    createUserHandler,  // Handler signature defines ParamTypeT, BodyTypeT, ResponseBodyT
    typed.WithRequestID,  // Types inferred! No need for WithRequestID[Params, Body, Response]
    typed.WithLogging,    // Types inferred! No need for WithLogging[Params, Body, Response]
)
```

**When explicit types ARE needed:**

Explicit type parameters are only needed when building middleware outside of `MakeHandler`:

```go
// Building middleware separately requires explicit types
middleware := typed.WithRequestID[Params, Body, Response]

// Then use it later
MakeHandler(Server, routeInfo, handler, middleware)
```

But in the common case (passing middleware directly to `MakeHandler`), **type inference works automatically**!

#### Best Practices

1. **Apply Early** - Add `http.WithRequestID()` as one of the first middleware in your router
2. **Always Log** - Include request IDs in all log statements for correlation
3. **Propagate Downstream** - Pass request IDs to all external service calls
4. **Client-Provided IDs** - Accept request IDs from clients for end-to-end tracing
5. **Unique Format** - Use UUIDs for uniqueness across distributed systems
6. **Use Type Inference** - No need for explicit type parameters in most cases

#### Integration with APM Tools

Request IDs work seamlessly with APM tools like:
- **DataDog** - Automatically correlates logs with traces
- **New Relic** - Links requests across distributed services
- **Sentry** - Groups errors by request
- **Elastic APM** - Traces requests through microservices

## Best Practices

1. **Use middleware composition** - Chain middleware in logical order (auth → validation → logging)
2. **Leverage generics** - Let the type system catch errors at compile time
3. **Safe Nullable access** - Use `Nullable.Value()` only when middleware guarantees it's set; use `HasValue()` or `TryValue()` for optional fields
4. **Structured logging** - Include context in all log messages
5. **Transaction safety** - Always use `db.WithTx()` for multi-statement operations
6. **Error details** - Provide meaningful error messages and field-level validation errors
7. **API documentation** - Fill in comprehensive RouteInfo for better Swagger docs

## Contributing

This library is extracted from a production API framework. Contributions are welcome!

## License

[Specify your license here]

## Support

For questions and support, please open an issue on the repository.
