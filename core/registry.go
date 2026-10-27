package core

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
)

// CodeInfo describes a registered error code.
type CodeInfo struct {
	// Title is the fixed, caller-facing summary written as the problem's
	// "title". It never varies per request.
	Title string
	// Status is the HTTP status the code is used with.
	Status int
	// Extensions lists the extension member keys a problem with this code
	// may carry. Any other key is dropped (and panics under `go test`).
	Extensions []string
}

// Registry resolves error codes. Implementations must be safe for
// concurrent use; Lookup is called for every error response.
type Registry interface {
	Lookup(code Code) (CodeInfo, bool)
}

// RegistryFunc adapts a function to Registry.
type RegistryFunc func(Code) (CodeInfo, bool)

// Lookup calls f.
func (f RegistryFunc) Lookup(c Code) (CodeInfo, bool) { return f(c) }

var (
	registry atomic.Pointer[registryHolder]
	typeBase atomic.Pointer[string]
)

type registryHolder struct{ r Registry }

// SetCodeRegistry installs the registry used to resolve codes. Call it once
// at start-up, before serving. Passing nil removes it.
//
// With a registry installed every code must resolve, through the registry or
// BuiltinCodes. A problem with an unknown code, a status that differs from
// the registered one, or an undeclared extension is a programming error: it
// panics under `go test` and, in production, an unknown code is written as
// the generic 500 internal_error (the others are corrected and logged).
//
// Without a registry any code is accepted and titles fall back to
// BuiltinCodes, then to the HTTP status text.
func SetCodeRegistry(r Registry) {
	if r == nil {
		registry.Store(nil)
		return
	}
	registry.Store(&registryHolder{r: r})
}

// SetTypeBase sets the prefix of every problem "type": the written type is
// base + code, e.g. "https://example.com/errors/" + "not_found". The base
// should resolve to documentation of each code. Call it once at start-up.
//
// When unset (or set to ""), "type" is "about:blank" and "title" is the HTTP
// status text, as RFC 9457 section 4.2.1 requires; the code is then not
// visible to clients, so services that publish codes must set a base.
func SetTypeBase(base string) {
	typeBase.Store(&base)
}

func currentTypeBase() string {
	if p := typeBase.Load(); p != nil {
		return *p
	}
	return ""
}

var builtinCodes = map[Code]CodeInfo{
	CodeBadRequest:           {Title: "Bad request", Status: http.StatusBadRequest},
	CodeUnauthorized:         {Title: "Unauthorized", Status: http.StatusUnauthorized},
	CodePermissionDenied:     {Title: "Permission denied", Status: http.StatusForbidden},
	CodeNotFound:             {Title: "Not found", Status: http.StatusNotFound},
	CodeConflict:             {Title: "Conflict", Status: http.StatusConflict},
	CodeRequestTooLarge:      {Title: "Request too large", Status: http.StatusRequestEntityTooLarge},
	CodeUnsupportedMediaType: {Title: "Unsupported media type", Status: http.StatusUnsupportedMediaType},
	CodeValidationFailed:     {Title: "Validation failed", Status: http.StatusUnprocessableEntity},
	CodeRateLimited:          {Title: "Too many requests", Status: http.StatusTooManyRequests},
	CodeInternalError:        {Title: "Internal server error", Status: http.StatusInternalServerError},
	CodeNotImplemented:       {Title: "Not implemented", Status: http.StatusNotImplemented},
	CodeUpstreamFailed:       {Title: "Upstream service failed", Status: http.StatusBadGateway},
	CodeUnavailable:          {Title: "Service unavailable", Status: http.StatusServiceUnavailable},
	CodeTimeout:              {Title: "Request timed out", Status: http.StatusGatewayTimeout},
	CodeInvalidJSON:          {Title: "Invalid JSON", Status: http.StatusBadRequest},
	CodeInvalidCSV:           {Title: "Invalid CSV", Status: http.StatusBadRequest},
	CodeFileTooLarge:         {Title: "File too large", Status: http.StatusRequestEntityTooLarge},
}

// BuiltinCodes returns the codes japi-core itself uses, with their default
// titles and statuses. A registry may override any of them; one that
// documents every code it serves should include all of them.
func BuiltinCodes() map[Code]CodeInfo {
	out := make(map[Code]CodeInfo, len(builtinCodes))
	for c, info := range builtinCodes {
		out[c] = info
	}
	return out
}

// lookupCode resolves c through the registry, then the built-in codes.
// registered reports whether a registry is installed.
func lookupCode(c Code) (info CodeInfo, found, registered bool) {
	if h := registry.Load(); h != nil {
		registered = true
		if info, ok := h.r.Lookup(c); ok {
			return info, true, true
		}
	}
	info, found = builtinCodes[c]
	return info, found, registered
}

// standardMembers are the RFC 9457 members and this package's standard
// extensions; an extension may never replace one of them.
var standardMembers = map[string]struct{}{
	"type": {}, "title": {}, "status": {}, "detail": {}, "instance": {},
	"trace_id": {}, "errors": {},
}

// ContractViolation is the panic value used, under `go test` only, when a
// problem breaks the registry contract (unknown code, status mismatch,
// undeclared or reserved extension key). Recovery middleware re-panics it so
// the test fails instead of observing a 500.
type ContractViolation string

func (c ContractViolation) Error() string { return "problem contract violation: " + string(c) }

// violation panics under `go test` and otherwise reports false so the caller
// applies its production fallback.
func violation(format string, args ...any) {
	if testing.Testing() {
		panic(ContractViolation(fmt.Sprintf(format, args...)))
	}
}

// sortedKeys returns the keys of m in order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// validCode reports whether c is a lowercase snake_case token.
func validCode(c Code) bool {
	if c == "" {
		return false
	}
	for _, r := range string(c) {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return !strings.HasPrefix(string(c), "_")
}
