// Package validation provides documentation and examples for implementing custom validators.
//
// This package demonstrates how to create custom validators that integrate with
// the go-playground/validator package. It does NOT provide pre-built validators
// with hardcoded business logic, as that would make the library non-reusable.
//
// Instead, this package shows you how to implement your own custom validators
// for your specific application needs.
package validation

// IMPORTANT: This package does NOT provide pre-built database validators
// because they would contain business-specific logic (table names, schemas, etc.)
// that won't work for other projects.
//
// Instead, use the examples below as templates to create your own custom
// validators specific to your application's database schema.

// Example: How to Register Custom Validators
//
// In your application's main.go or initialization code:
//
//	import (
//	    "database/sql"
//	    "github.com/go-playground/validator/v10"
//	)
//
//	func setupValidators(db *sql.DB) *validator.Validate {
//	    validate := validator.New()
//
//	    // Register custom database-backed validators
//	    validate.RegisterValidation("unique_email", uniqueEmailValidator(db))
//	    validate.RegisterValidation("user_exists", userExistsValidator(db))
//
//	    return validate
//	}
//
//	// Example: Unique email validator
//	func uniqueEmailValidator(db *sql.DB) validator.Func {
//	    return func(fl validator.FieldLevel) bool {
//	        email := fl.Field().String()
//	        if email == "" {
//	            return true // Let 'required' tag handle empty values
//	        }
//
//	        var count int
//	        // REPLACE 'users' with your actual table name
//	        err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
//	        if err != nil {
//	            // Log the error in production
//	            return false
//	        }
//	        return count == 0 // Valid if email doesn't exist
//	    }
//	}
//
//	// Example: User exists validator
//	func userExistsValidator(db *sql.DB) validator.Func {
//	    return func(fl validator.FieldLevel) bool {
//	        userID := fl.Field().String() // Or .Int() depending on your ID type
//	        if userID == "" {
//	            return false
//	        }
//
//	        var exists bool
//	        // REPLACE 'users' with your actual table name
//	        err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
//	        if err != nil {
//	            return false
//	        }
//	        return exists
//	    }
//	}
//
// Then use in your structs:
//
//	type CreateUserRequest struct {
//	    Email string `json:"email" validate:"required,email,unique_email"`
//	}
//
//	type AssignTaskRequest struct {
//	    UserID string `json:"user_id" validate:"required,uuid,user_exists"`
//	}

// ValidatorSetup is a helper type for organizing custom validators.
//
// You can use this pattern in your application to keep validators organized:
//
//	type ValidatorSetup struct {
//	    db       *sql.DB
//	    validate *validator.Validate
//	}
//
//	func NewValidatorSetup(db *sql.DB) *ValidatorSetup {
//	    return &ValidatorSetup{
//	        db:       db,
//	        validate: validator.New(),
//	    }
//	}
//
//	func (vs *ValidatorSetup) RegisterAll() *validator.Validate {
//	    vs.registerDatabaseValidators()
//	    vs.registerBusinessRules()
//	    return vs.validate
//	}
//
//	func (vs *ValidatorSetup) registerDatabaseValidators() {
//	    vs.validate.RegisterValidation("unique_email", vs.uniqueEmail())
//	    vs.validate.RegisterValidation("user_exists", vs.userExists())
//	}
//
//	func (vs *ValidatorSetup) registerBusinessRules() {
//	    // Add your business rule validators here
//	}
type ValidatorSetup struct {
	// This is just a documentation example.
	// Implement this pattern in your own application code.
}

// Common Validator Patterns
//
// 1. Uniqueness Validators
//    - Check if a value is unique in a database table
//    - Use COUNT(*) or EXISTS for efficiency
//    - Remember to handle empty values appropriately
//
// 2. Existence Validators
//    - Verify that a foreign key reference exists
//    - Use EXISTS subquery for best performance
//    - Return false for invalid/missing IDs
//
// 3. Business Rule Validators
//    - Complex validation requiring database queries
//    - Can query multiple tables
//    - Should be fast to avoid slowing down requests
//
// 4. Cross-Field Validators
//    - Compare multiple fields in the same struct
//    - Use fl.Parent() to access other fields
//
// Best Practices:
//
// - Keep validators fast (add database indexes)
// - Log validation errors for debugging
// - Use prepared statements if validating in loops
// - Consider caching for frequently validated values
// - Handle database errors gracefully (assume invalid on error)
// - Use context with timeouts for database queries
// - Don't put expensive operations in validators

// Example: Using Context in Validators
//
// For production validators, use context with timeouts:
//
//	func uniqueEmailWithContext(db *sql.DB) validator.Func {
//	    return func(fl validator.FieldLevel) bool {
//	        email := fl.Field().String()
//	        if email == "" {
//	            return true
//	        }
//
//	        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
//	        defer cancel()
//
//	        var count int
//	        err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
//	        if err != nil {
//	            // Log error in production
//	            return false
//	        }
//	        return count == 0
//	    }
//	}

// Example: Custom Error Messages
//
// When using custom validators, update the error message generator in your
// middleware/typed/request.go or create a custom one:
//
//	func generateFieldErrorMessage(fieldError validator.FieldError) string {
//	    fieldName := fieldError.Field()
//	    tag := fieldError.Tag()
//	    param := fieldError.Param()
//
//	    switch tag {
//	    case "unique_email":
//	        return "A user with this email already exists"
//	    case "user_exists":
//	        return "User does not exist"
//	    case "valid_status":
//	        return fmt.Sprintf("Status must be one of: %s", param)
//	    default:
//	        return fmt.Sprintf("%s validation failed", fieldName)
//	    }
//	}

// For more information on custom validators, see:
// https://pkg.go.dev/github.com/go-playground/validator/v10#Validate.RegisterValidation
