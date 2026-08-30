package core

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueConstraintError checks if an error is a unique constraint violation for a specific constraint
func isUniqueConstraintError(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	// PostgreSQL unique constraint violation error code is "23505"
	if pgErr.Code != "23505" {
		return false
	}

	// Check if the error message contains the specific constraint name
	return strings.Contains(pgErr.ConstraintName, constraintName)
}

// IsUniqueConstraintError is a public wrapper for constraint error checking
func IsUniqueConstraintError(err error, constraintName string) bool {
	return isUniqueConstraintError(err, constraintName)
}

// isForeignKeyConstraintError checks if an error is a foreign key constraint violation for a specific constraint
func isForeignKeyConstraintError(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	// PostgreSQL foreign key constraint violation error code is "23503"
	if pgErr.Code != "23503" {
		return false
	}

	// Check if the error message contains the specific constraint name
	return strings.Contains(pgErr.ConstraintName, constraintName)
}

// IsForeignKeyConstraintError is a public wrapper for foreign key constraint error checking
func IsForeignKeyConstraintError(err error, constraintName string) bool {
	return isForeignKeyConstraintError(err, constraintName)
}
