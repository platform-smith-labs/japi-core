package core

import (
	"net/http"
)

// Code is a machine-readable error code: a lowercase snake_case token such as
// "not_found" or "validation_failed". It is the last path segment of a
// problem's "type" URI and the span's error.type, so it must be a constant
// chosen by the program, never text derived from a request or another error.
type Code string

// Built-in codes. japi-core constructs these itself (request parsing,
// authentication, the fallback for unexpected errors, panics). A registry set
// with SetCodeRegistry may override their titles; see BuiltinCodes.
const (
	CodeBadRequest           Code = "bad_request"
	CodeUnauthorized         Code = "unauthorized"
	CodePermissionDenied     Code = "permission_denied"
	CodeNotFound             Code = "not_found"
	CodeMethodNotAllowed     Code = "method_not_allowed"
	CodeConflict             Code = "conflict"
	CodeRequestTooLarge      Code = "request_too_large"
	CodeUnsupportedMediaType Code = "unsupported_media_type"
	CodeValidationFailed     Code = "validation_failed"
	CodeRateLimited          Code = "rate_limited"
	CodeInternalError        Code = "internal_error"
	CodeNotImplemented       Code = "not_implemented"
	CodeUpstreamFailed       Code = "upstream_failed"
	CodeUnavailable          Code = "unavailable"
	CodeTimeout              Code = "timeout"

	// Request-body classes used by the upload and body parsing middleware.
	CodeInvalidJSON  Code = "invalid_json"
	CodeInvalidCSV   Code = "invalid_csv"
	CodeFileTooLarge Code = "file_too_large"
)

// FieldError is one entry of a problem's "errors" member.
type FieldError struct {
	// Detail is a caller-facing description of what is wrong with the field.
	// It must not quote the rejected value.
	Detail string `json:"detail"`
	// Pointer is a JSON Pointer in URI fragment form ("#/name",
	// "#/items/0/sku") into the JSON request body. Leave it empty for query
	// or path parameters and name the parameter in Detail instead.
	Pointer string `json:"pointer,omitempty"`
}

// Problem is the only error type japi-core writes to a client, as an RFC 9457
// Problem Details body (application/problem+json).
//
// Return a *Problem from a handler (directly or wrapped with %w; handlers
// find it with errors.As). Any other error becomes the generic 500
// internal_error problem.
//
// What reaches the client:
//   - "type" is the type base (SetTypeBase) followed by Code;
//   - "title" is the code's registered title, never request-specific;
//   - "status" is Status;
//   - "detail" is Detail for a 4xx. For a 5xx Detail is never written: the
//     title is the whole message, and Detail goes to the log only;
//   - "trace_id" is the trace ID of the request's span;
//   - "errors" is Errors (4xx only);
//   - Extensions are added as top-level members (see WithExtension).
//
// Cause is never written to the client. It is logged and, for a 5xx,
// recorded on the span.
type Problem struct {
	Code       Code
	Status     int
	Detail     string
	Cause      error
	Errors     []FieldError
	Extensions map[string]any
}

// Error returns the code and, when present, the detail. The cause is not
// included; reach it with errors.Unwrap.
func (p *Problem) Error() string {
	if p == nil {
		return "<nil problem>"
	}
	s := string(p.Code)
	if s == "" {
		s = "problem"
	}
	if p.Detail != "" {
		s += ": " + p.Detail
	}
	return s
}

// Unwrap returns the cause.
func (p *Problem) Unwrap() error { return p.Cause }

// WithCause sets the cause and returns p.
func (p *Problem) WithCause(err error) *Problem {
	p.Cause = err
	return p
}

// WithDetail sets the detail and returns p.
func (p *Problem) WithDetail(detail string) *Problem {
	p.Detail = detail
	return p
}

// WithErrors appends field errors and returns p.
func (p *Problem) WithErrors(errs ...FieldError) *Problem {
	p.Errors = append(p.Errors, errs...)
	return p
}

// WithExtension adds a top-level extension member and returns p. Use
// lowercase snake_case keys. A key that names a standard member ("type",
// "title", "status", "detail", "instance", "trace_id", "errors") is never
// written; when a code registry is set, only the extension keys it declares
// for the code are written. Both cases panic under `go test`.
func (p *Problem) WithExtension(key string, value any) *Problem {
	if p.Extensions == nil {
		p.Extensions = make(map[string]any)
	}
	p.Extensions[key] = value
	return p
}

// NewProblem returns a problem with the given code, status and caller-facing
// detail. A zero status takes the code's registered status.
func NewProblem(code Code, status int, detail string) *Problem {
	return &Problem{Code: code, Status: status, Detail: detail}
}

// BadRequest returns a 400 problem.
func BadRequest(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusBadRequest, detail)
}

// Unauthorized returns a 401 problem.
func Unauthorized(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusUnauthorized, detail)
}

// Forbidden returns a 403 problem.
func Forbidden(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusForbidden, detail)
}

// NotFound returns a 404 problem.
func NotFound(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusNotFound, detail)
}

// Conflict returns a 409 problem.
func Conflict(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusConflict, detail)
}

// Unprocessable returns a 422 problem.
func Unprocessable(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusUnprocessableEntity, detail)
}

// TooManyRequests returns a 429 problem.
func TooManyRequests(code Code, detail string) *Problem {
	return NewProblem(code, http.StatusTooManyRequests, detail)
}

// Validation returns the 422 validation_failed problem carrying errs.
func Validation(errs ...FieldError) *Problem {
	return NewProblem(CodeValidationFailed, http.StatusUnprocessableEntity, "").WithErrors(errs...)
}

// Internal returns the generic 500 internal_error problem with cause.
func Internal(cause error) *Problem {
	return NewProblem(CodeInternalError, http.StatusInternalServerError, "").WithCause(cause)
}

// BadGateway returns a 502 problem with cause. The client sees only the
// code's title.
func BadGateway(code Code, cause error) *Problem {
	return NewProblem(code, http.StatusBadGateway, "").WithCause(cause)
}

// Unavailable returns a 503 problem with cause. The client sees only the
// code's title.
func Unavailable(code Code, cause error) *Problem {
	return NewProblem(code, http.StatusServiceUnavailable, "").WithCause(cause)
}

// GatewayTimeout returns a 504 problem with cause. The client sees only the
// code's title.
func GatewayTimeout(code Code, cause error) *Problem {
	return NewProblem(code, http.StatusGatewayTimeout, "").WithCause(cause)
}
