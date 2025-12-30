package ratelimit

import (
	"strings"
)

// MatchEndpoint matches a request path and method to an endpoint configuration.
// Returns the matching EndpointConfig or nil if no match is found.
// Path matching supports prefix matching (e.g., "/runs/" matches "/runs/{id}").
func MatchEndpoint(path string, method string, configs []EndpointConfig) *EndpointConfig {
	// Special case: health check endpoint is unlimited
	if path == "/health" && method == "GET" {
		return &EndpointConfig{
			Limit:  0, // Unlimited
			Window: 0,
			Burst:  0,
		}
	}

	// Try exact match first
	for i := range configs {
		config := &configs[i]
		if config.Path == path && config.Method == method {
			return config
		}
	}

	// Try prefix match (for paths ending with "/")
	for i := range configs {
		config := &configs[i]
		if config.Method == method && strings.HasSuffix(config.Path, "/") {
			if strings.HasPrefix(path, config.Path) {
				return config
			}
		}
	}

	// No match found
	return nil
}
