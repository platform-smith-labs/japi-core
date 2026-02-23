# Migration Guide: v1.x to v2.0.0

This guide helps you migrate from japi-core v1.x to v2.0.0, which introduces proper `context.Context` propagation throughout the framework.

## Overview

Version 2.0.0 is a **breaking change** that adds context propagation support. This enables:

- ✅ Request cancellation when clients disconnect
- ✅ Timeout propagation from HTTP requests to database queries
- ✅ Distributed tracing support
- ✅ Prevention of resource leaks
- ✅ Production-ready cancellation handling

## Breaking Changes

### 1. Database Function Signatures

All database functions now require `context.Context` as the **first parameter**.

#### QueryMany

```go
// v1.x
users, err := db.QueryMany[User](ctx.DB,
    "SELECT * FROM users WHERE status = $1",
    "active")

// v2.0.0
users, err := db.QueryMany[User](ctx.Context, ctx.DB,  // Added ctx.Context
    "SELECT * FROM users WHERE status = $1",
    "active")
```

#### QueryOne

```go
// v1.x
user, err := db.QueryOne[User](ctx.DB,
    "SELECT * FROM users WHERE id = $1",
    userID)

// v2.0.0
user, err := db.QueryOne[User](ctx.Context, ctx.DB,  // Added ctx.Context
    "SELECT * FROM users WHERE id = $1",
    userID)
```

#### Exec

```go
// v1.x
result, err := db.Exec(ctx.DB,
    "UPDATE users SET name = $1 WHERE id = $2",
    newName, userID)

// v2.0.0
result, err := db.Exec(ctx.Context, ctx.DB,  // Added ctx.Context
    "UPDATE users SET name = $1 WHERE id = $2",
    newName, userID)
```

#### WithTx (Transaction)

```go
// v1.x
err := db.WithTx(ctx.DB, func(tx *sql.Tx) error {
    _, err := db.Exec(tx, "INSERT INTO ...", ...)
    if err != nil {
        return err
    }
    _, err = db.Exec(tx, "UPDATE ...", ...)
    return err
})

// v2.0.0
err := db.WithTx(ctx.Context, ctx.DB, func(txCtx context.Context, tx *sql.Tx) error {
    _, err := db.Exec(txCtx, tx, "INSERT INTO ...", ...)  // Use txCtx
    if err != nil {
        return err
    }
    _, err = db.Exec(txCtx, tx, "UPDATE ...", ...)  // Use txCtx
    return err
})
```

**Note**: The transaction callback now receives both `context.Context` and `*sql.Tx`.

### 2. HandlerContext

`HandlerContext` now includes a `Context` field that holds the HTTP request context:

```go
type HandlerContext[ParamTypeT any, BodyTypeT any] struct {
    Context context.Context  // NEW: Request context
    DB      *sql.DB
    Logger  *slog.Logger
    // ... other fields
}
```

**Important**: The adapter automatically sets `Context` from `r.Context()`. You don't need to manually set it in handlers.

### 3. Context Error Handling

The adapter now handles context-specific errors:

- `context.Canceled`: Client disconnected (logs and returns without writing response)
- `context.DeadlineExceeded`: Request timeout (returns 504 Gateway Timeout)

You can check for these errors in your handlers:

```go
func GetUsers(ctx handler.HandlerContext[...]) (*Response, error) {
    users, err := db.QueryMany[User](ctx.Context, ctx.DB, ...)
    if err != nil {
        if errors.Is(err, context.Canceled) {
            return nil, core.NewAPIError(499, "Client closed request")
        }
        if errors.Is(err, context.DeadlineExceeded) {
            return nil, core.NewAPIError(504, "Request timeout")
        }
        return nil, core.ErrInternal(err.Error())
    }
    return &Response{Users: users}, nil
}
```

## Migration Steps

### Step 1: Update Imports

Add `context` and `errors` to your imports if handling context errors:

```go
import (
    "context"
    "errors"
    "github.com/platform-smith-labs/japi-core/core"
    "github.com/platform-smith-labs/japi-core/handler"
    "github.com/platform-smith-labs/japi-core/db"
)
```

### Step 2: Find All Database Calls

Search your codebase for:
- `db.QueryMany`
- `db.QueryOne`
- `db.Exec`
- `db.WithTx`

### Step 3: Update Each Call

For each database call, add `ctx.Context` as the first parameter:

**Before**:
```go
func GetUser(ctx handler.HandlerContext[GetUserParams, GetUserRequest]) (*GetUserResponse, error) {
    params := ctx.Params.Value()

    user, err := db.QueryOne[User](ctx.DB,
        "SELECT * FROM users WHERE id = $1",
        params.UserID)

    if err == sql.ErrNoRows {
        return nil, core.ErrNotFound("User not found")
    }
    if err != nil {
        return nil, core.ErrInternal(err.Error())
    }

    return &GetUserResponse{User: user}, nil
}
```

**After**:
```go
func GetUser(ctx handler.HandlerContext[GetUserParams, GetUserRequest]) (*GetUserResponse, error) {
    params := ctx.Params.Value()

    user, err := db.QueryOne[User](ctx.Context, ctx.DB,  // Added ctx.Context
        "SELECT * FROM users WHERE id = $1",
        params.UserID)

    if err == sql.ErrNoRows {
        return nil, core.ErrNotFound("User not found")
    }
    if err != nil {
        return nil, core.ErrInternal(err.Error())
    }

    return &GetUserResponse{User: user}, nil
}
```

### Step 4: Update Transaction Code

For transactions, update both the `WithTx` call and the callback signature:

**Before**:
```go
err := db.WithTx(ctx.DB, func(tx *sql.Tx) error {
    // Insert user
    var userID int
    err := tx.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id",
        name).Scan(&userID)
    if err != nil {
        return err
    }

    // Create profile
    _, err = db.Exec(tx, "INSERT INTO profiles (user_id) VALUES ($1)", userID)
    return err
})
```

**After**:
```go
err := db.WithTx(ctx.Context, ctx.DB, func(txCtx context.Context, tx *sql.Tx) error {
    // Insert user
    var userID int
    err := tx.QueryRowContext(txCtx, "INSERT INTO users (name) VALUES ($1) RETURNING id",
        name).Scan(&userID)
    if err != nil {
        return err
    }

    // Create profile
    _, err = db.Exec(txCtx, tx, "INSERT INTO profiles (user_id) VALUES ($1)", userID)
    return err
})
```

### Step 5: Update Tests

Update your test code to set `Context` in HandlerContext:

**Before**:
```go
func TestGetUser(t *testing.T) {
    ctx := handler.HandlerContext[GetUserParams, GetUserRequest]{
        DB:     testDB,
        Logger: slog.Default(),
    }
    // ... test code
}
```

**After**:
```go
func TestGetUser(t *testing.T) {
    ctx := handler.HandlerContext[GetUserParams, GetUserRequest]{
        Context: context.Background(),  // Added
        DB:      testDB,
        Logger:  slog.Default(),
    }
    // ... test code
}
```

For timeout tests:
```go
func TestGetUserTimeout(t *testing.T) {
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    ctx := handler.HandlerContext[GetUserParams, GetUserRequest]{
        Context: timeoutCtx,  // Test timeout
        DB:      testDB,
        Logger:  slog.Default(),
    }
    // ... test that timeout is handled
}
```

### Step 6: Compile and Test

```bash
# Compile
go build ./...

# Run tests
go test ./... -v

# If compilation fails, check error messages for missed database calls
```

## Automated Migration

You can use this shell script to help find calls that need updating:

```bash
#!/bin/bash

echo "Finding database calls that need ctx.Context parameter..."
echo ""

echo "QueryMany calls:"
grep -rn "db.QueryMany\[" --include="*.go" | grep -v "ctx.Context"

echo ""
echo "QueryOne calls:"
grep -rn "db.QueryOne\[" --include="*.go" | grep -v "ctx.Context"

echo ""
echo "Exec calls:"
grep -rn "db.Exec(" --include="*.go" | grep -v "ctx.Context"

echo ""
echo "WithTx calls:"
grep -rn "db.WithTx" --include="*.go" | grep -v "ctx.Context"
```

## Common Patterns

### Pattern 1: Simple Handler

```go
// v2.0.0
func ListUsers(ctx handler.HandlerContext[ListUsersParams, ListUsersRequest]) (*ListUsersResponse, error) {
    users, err := db.QueryMany[User](ctx.Context, ctx.DB,
        "SELECT id, name, email FROM users ORDER BY name")
    if err != nil {
        return nil, core.ErrInternal(err.Error())
    }
    return &ListUsersResponse{Users: users}, nil
}
```

### Pattern 2: Handler with Transaction

```go
// v2.0.0
func CreateOrder(ctx handler.HandlerContext[CreateOrderParams, CreateOrderRequest]) (*CreateOrderResponse, error) {
    req := ctx.Body.Value()

    var orderID int
    err := db.WithTx(ctx.Context, ctx.DB, func(txCtx context.Context, tx *sql.Tx) error {
        // Create order
        err := tx.QueryRowContext(txCtx,
            "INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id",
            req.UserID, req.Total).Scan(&orderID)
        if err != nil {
            return err
        }

        // Create order items
        for _, item := range req.Items {
            _, err := db.Exec(txCtx, tx,
                "INSERT INTO order_items (order_id, product_id, quantity) VALUES ($1, $2, $3)",
                orderID, item.ProductID, item.Quantity)
            if err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, core.ErrInternal(err.Error())
    }

    return &CreateOrderResponse{OrderID: orderID}, nil
}
```

### Pattern 3: Handler with Cancellation Check

```go
// v2.0.0
func ExportUsers(ctx handler.HandlerContext[ExportUsersParams, ExportUsersRequest]) (*ExportUsersResponse, error) {
    // Large query that might be cancelled
    users, err := db.QueryMany[User](ctx.Context, ctx.DB,
        "SELECT * FROM users")

    if err != nil {
        // Check if cancelled
        if errors.Is(err, context.Canceled) {
            ctx.Logger.Info("Export cancelled by client")
            return nil, core.NewAPIError(499, "Export cancelled")
        }
        // Check if timeout
        if errors.Is(err, context.DeadlineExceeded) {
            ctx.Logger.Error("Export timeout")
            return nil, core.NewAPIError(504, "Export timeout")
        }
        return nil, core.ErrInternal(err.Error())
    }

    // Process large result set
    data := processUsers(users)
    return &ExportUsersResponse{Data: data}, nil
}
```

## Benefits After Migration

Once migrated, your application will benefit from:

1. **Automatic Cancellation**: Database queries stop when clients disconnect
2. **Timeout Support**: Queries respect request-level timeouts
3. **Resource Efficiency**: No wasted database connections or CPU cycles
4. **Better Observability**: Context can carry trace IDs for distributed tracing
5. **Production Reliability**: Handles edge cases like slow clients and network issues

## Troubleshooting

### Error: "not enough arguments in call to db.QueryMany"

**Cause**: Missing `ctx.Context` parameter

**Fix**: Add `ctx.Context` as the first parameter:
```go
// Before
users, err := db.QueryMany[User](ctx.DB, query)

// After
users, err := db.QueryMany[User](ctx.Context, ctx.DB, query)
```

### Error: "cannot use func literal (type func(*sql.Tx) error) as type func(context.Context, *sql.Tx) error"

**Cause**: Transaction callback signature is old

**Fix**: Update callback to accept both context and transaction:
```go
// Before
db.WithTx(ctx.Context, ctx.DB, func(tx *sql.Tx) error {
    // ...
})

// After
db.WithTx(ctx.Context, ctx.DB, func(txCtx context.Context, tx *sql.Tx) error {
    // ...
})
```

### Tests Fail with nil Context

**Cause**: Test HandlerContext doesn't have Context set

**Fix**: Always set Context in tests:
```go
ctx := handler.HandlerContext[...]{
    Context: context.Background(),  // Add this
    DB:      testDB,
    Logger:  slog.Default(),
}
```

## Support

If you encounter issues during migration:

1. Check this guide for common patterns
2. Review the examples in README.md
3. Open an issue on GitHub: https://github.com/platform-smith-labs/japi-core/issues

## Version History

- **v2.0.0**: Added context propagation (breaking change)
- **v1.0.0**: Initial release
