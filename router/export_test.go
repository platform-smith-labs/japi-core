package router

// DefaultAllowedMethods is a test helper that exposes the default CORS allowed
// methods list from defaultRouterConfig. Only compiled during tests.
func DefaultAllowedMethods() []string {
	cfg := defaultRouterConfig()
	result := make([]string, len(cfg.allowedMethods))
	copy(result, cfg.allowedMethods)
	return result
}
