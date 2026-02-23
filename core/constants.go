package core

// Standard HTTP status codes for database operations
const (
	// StatusDatabaseConstraintViolation is used for all database constraint violations
	// (foreign key, unique, check constraints)
	StatusDatabaseConstraintViolation = 400

	// StatusDatabaseError is used for all other database errors
	StatusDatabaseError = 500
)
