package handler

import (
	"net/http"

	"github.com/platform-smith-labs/japi-core/core"
)

// Nullable represents an optional value that may or may not contain data.
//
// It provides explicit optionality for request-scoped data in HandlerContext,
// such as parameters, body, headers, and authentication information.
//
// Design Philosophy:
//
// Nullable implements explicit error handling for optional values. Middleware is
// responsible for populating values, and handlers must check for errors when
// calling Value(). This provides idiomatic Go error handling while maintaining
// type safety for request-scoped data.
//
// Example Usage:
//
//	// Middleware populates the value
//	ctx.Body = handler.NewNullable(validatedBody)
//
//	// Handler checks for errors when accessing value
//	func CreateUser(ctx HandlerContext) (*Response, error) {
//	    body, err := ctx.Body.Value() // Returns error if middleware didn't run
//	    if err != nil {
//	        return nil, core.ErrInternal("missing request body")
//	    }
//	    // Use body safely...
//	}
//
// For optional values that may legitimately be absent, use HasValue() or TryValue():
//
//	if ctx.UserUUID.HasValue() {
//	    userID, _ := ctx.UserUUID.Value() // Safe after HasValue() check
//	    // User is authenticated
//	}
type Nullable[T any] struct {
	value    T
	hasValue bool
}

// NewNullable creates a Nullable containing the given value.
func NewNullable[T any](value T) Nullable[T] {
	return Nullable[T]{
		value:    value,
		hasValue: true,
	}
}

// Nil returns an empty Nullable with no value.
func Nil[T any]() Nullable[T] {
	return Nullable[T]{
		hasValue: false,
	}
}

// HasValue returns true if the Nullable contains a value.
//
// Use this method when you need to check if a value is present before
// accessing it, particularly for optional fields like authentication data.
func (n Nullable[T]) HasValue() bool {
	return n.hasValue
}

// Value returns the contained value and an error if no value is present.
//
// This method provides idiomatic Go error handling for accessing nullable values.
// It returns an error if the Nullable is empty, allowing callers to handle the
// absence explicitly.
//
// Use Value() when you need explicit error handling at the call site. For optional
// values where you want to provide a default, use ValueOr() or ValueOrDefault().
// For simple presence checks, use HasValue() or TryValue().
//
// Returns an error if HasValue() is false.
//
// Example:
//
//	body, err := ctx.Body.Value()
//	if err != nil {
//	    return nil, core.ErrInternal("missing request body")
//	}
func (n Nullable[T]) Value() (T, error) {
	if !n.hasValue {
		var zero T
		return zero, core.NewAPIError(http.StatusInternalServerError, "nullable value is not present")
	}
	return n.value, nil
}

// TryValue returns the contained value and a boolean indicating whether the value exists.
//
// This is the safe way to access a Nullable value when you're not certain it's present.
// It never panics and is useful for middleware or conditional logic.
//
// Example:
//
//	if userID, ok := ctx.UserUUID.TryValue(); ok {
//	    // User is authenticated, use userID
//	} else {
//	    // User is not authenticated
//	}
func (n Nullable[T]) TryValue() (T, bool) {
	return n.value, n.hasValue
}

// ValueOrDefault returns the value if present, otherwise returns the zero value for type T.
//
// The zero value depends on the type:
//   - 0 for numeric types
//   - "" for strings
//   - false for bool
//   - nil for pointers/slices/maps
//   - empty struct for struct types
//
// Example:
//
//	sort := ctx.Params.ValueOrDefault() // Returns empty struct if no params
func (n Nullable[T]) ValueOrDefault() T {
	if n.hasValue {
		return n.value
	}
	var zero T
	return zero
}

// ValueOr returns the value if present, otherwise returns the provided default value.
//
// Use this when you have a specific fallback value in mind.
//
// Example:
//
//	limit := ctx.Limit.ValueOr(10) // Default to 10 if not specified
func (n Nullable[T]) ValueOr(defaultValue T) T {
	if n.hasValue {
		return n.value
	}
	return defaultValue
}
