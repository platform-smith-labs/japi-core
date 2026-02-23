package handler

// Nullable represents an optional value that may or may not contain data.
//
// It provides explicit optionality for request-scoped data in HandlerContext,
// such as parameters, body, headers, and authentication information.
//
// Design Philosophy:
//
// Nullable implements a fail-fast pattern where middleware is responsible for
// populating values, and handlers can safely assume the data is present by
// calling Value(). This eliminates defensive nil checks in business logic
// when middleware contracts are properly enforced.
//
// Example Usage:
//
//	// Middleware populates the value
//	ctx.Body = handler.NewNullable(validatedBody)
//
//	// Handler assumes middleware ran correctly
//	func CreateUser(ctx HandlerContext) (*Response, error) {
//	    body := ctx.Body.Value() // Panics if middleware didn't run
//	    // Use body safely...
//	}
//
// For optional values that may legitimately be absent, use HasValue() or TryValue():
//
//	if ctx.UserUUID.HasValue() {
//	    userID := ctx.UserUUID.Value()
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

// Value returns the contained value or panics if no value is present.
//
// This method implements fail-fast behavior: it assumes middleware has properly
// populated the value. If Value() is called on an empty Nullable, it panics
// with a recoverable panic (not os.Exit), allowing the application to continue
// running and the panic to be caught by recovery middleware.
//
// Use Value() when middleware guarantees the value is set (e.g., ParseBody
// middleware always populates ctx.Body for handlers that require it).
//
// For optional values, use HasValue() or TryValue() instead.
//
// Panics if HasValue() is false.
func (n Nullable[T]) Value() T {
	if !n.hasValue {
		panic("japi-core: attempted to access Nullable value when HasValue is false")
	}
	return n.value
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
