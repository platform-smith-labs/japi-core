package typed

import (
	"net/http"
	"strings"

	"github.com/platform-smith-labs/japi-core/core"
	"github.com/platform-smith-labs/japi-core/handler"
	"github.com/platform-smith-labs/japi-core/jwt"
	"github.com/google/uuid"
)

// RequireAuth validates JWT token from Authorization header and verifies user/company existence.
//
// This middleware extracts and validates the JWT token from the Authorization header,
// verifies that the user and company still exist in the database, and sets UserUUID
// and CompanyUUID in the context if valid.
//
// Parameters:
//   - jwtSecret: The secret key for validating JWT tokens
//   - validateUserCompany: Function to validate that user and company exist in database
//     The function should accept (querier, userUUID, companyUUID) and return error
//
// Dependencies: jwt package, database access via ctx.DB
// Context modifications: Sets ctx.UserUUID and ctx.CompanyUUID
// Use: Apply via MakeHandler(..., RequireAuth(secret, validator, ...), ...)
//
// Returns:
//   - 401 if Authorization header is missing, malformed, or token is invalid/expired
//   - 403/500 if user/company validation fails
//
// Example:
//
//	validateFunc := func(db interface{}, userID, companyID uuid.UUID) error {
//	    // Check if user and company exist in database
//	    return nil
//	}
//	handler := MakeHandler(myHandler, RequireAuth(jwtSecret, validateFunc, next), ResponseJSON)
func RequireAuth[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	jwtSecret string,
	validateUserCompany func(querier interface{}, userUUID, companyUUID uuid.UUID) error,
	next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			var zeroResponse ResponseBodyT
			return zeroResponse, core.NewAPIError(http.StatusUnauthorized, "Authorization header required")
		}

		// Check Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			var zeroResponse ResponseBodyT
			return zeroResponse, core.NewAPIError(http.StatusUnauthorized, "Authorization header must start with 'Bearer '")
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			var zeroResponse ResponseBodyT
			return zeroResponse, core.NewAPIError(http.StatusUnauthorized, "Bearer token is required")
		}

		// Validate JWT secret is provided
		if jwtSecret == "" {
			var zeroResponse ResponseBodyT
			ctx.Logger.Error("JWT secret not configured")
			return zeroResponse, core.NewAPIError(http.StatusInternalServerError, "Authentication configuration error")
		}

		// Validate JWT token
		claims, err := jwt.ValidateToken(token, jwtSecret)
		if err != nil {
			var zeroResponse ResponseBodyT
			ctx.Logger.Warn("Invalid JWT token", "error", err.Error())
			return zeroResponse, core.NewAPIError(http.StatusUnauthorized, "Invalid or expired token")
		}

		// Verify user and company still exist in database
		if err := validateUserCompany(ctx.DB, claims.UserUUID, claims.CompanyUUID); err != nil {
			var zeroResponse ResponseBodyT
			// This returns 403 for user/company not found or 500 for DB errors
			return zeroResponse, err
		}

		// Set authenticated user data in context
		ctx.UserUUID = handler.NewNullable(claims.UserUUID)
		ctx.CompanyUUID = handler.NewNullable(claims.CompanyUUID)

		// Call next handler with authenticated context
		return next(ctx, w, r)
	}
}
