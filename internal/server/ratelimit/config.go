package ratelimit

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EndpointConfig represents rate limiting configuration for a specific endpoint.
type EndpointConfig struct {
	Path   string        // Endpoint path pattern (supports prefix matching)
	Method string        // HTTP method (GET, POST, etc.)
	Limit  int           // Maximum requests per window
	Window time.Duration // Time window
	Burst  int           // Burst capacity (defaults to Limit if 0)
}

// LoadConfig loads rate limiting configuration from environment variables.
func LoadConfig() *Config {
	enabled := getEnvBool("RATE_LIMIT_ENABLED", true)
	if !enabled {
		return &Config{
			Enabled: false,
		}
	}

	defaultLimit := getEnvInt("RATE_LIMIT_DEFAULT_LIMIT", 1000)
	defaultWindow := getEnvDuration("RATE_LIMIT_DEFAULT_WINDOW", time.Minute)
	cleanupInterval := getEnvDuration("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute)

	whitelist := parseIPList(getEnvString("RATE_LIMIT_WHITELIST", ""))
	blacklist := parseIPList(getEnvString("RATE_LIMIT_BLACKLIST", ""))

	return &Config{
		Enabled:         enabled,
		DefaultLimit:    defaultLimit,
		DefaultWindow:   defaultWindow,
		CleanupInterval: cleanupInterval,
		Whitelist:       whitelist,
		Blacklist:       blacklist,
		EndpointConfigs: DefaultEndpointConfigs(),
	}
}

// DefaultEndpointConfigs returns the default endpoint-specific configurations.
func DefaultEndpointConfigs() []EndpointConfig {
	return []EndpointConfig{
		// Tier 1: Expensive operations (strictest limits)
		{Path: "/run", Method: "POST", Limit: 10, Window: time.Hour, Burst: 2},
		{Path: "/run/stream", Method: "POST", Limit: 10, Window: time.Hour, Burst: 2},
		{Path: "/runs/", Method: "POST", Limit: 10, Window: time.Hour, Burst: 2},

		// Tier 2: Write operations (moderate limits)
		{Path: "/users", Method: "POST", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/users/", Method: "PUT", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/users/", Method: "DELETE", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/jobs/", Method: "POST", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/jobs/", Method: "PUT", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/jobs/", Method: "DELETE", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/experiences/", Method: "POST", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/experiences/", Method: "PUT", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/experiences/", Method: "DELETE", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/education/", Method: "POST", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/education/", Method: "PUT", Limit: 100, Window: time.Minute, Burst: 10},
		{Path: "/education/", Method: "DELETE", Limit: 100, Window: time.Minute, Burst: 10},

		// Tier 3: Read operations (more lenient) - handled by default limit
		// Tier 4: Health check (unlimited) - handled by special case in matcher
	}
}

// getEnvString gets an environment variable as a string with a default value.
func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer with a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets an environment variable as a boolean with a default value.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvDuration gets an environment variable as a duration with a default value.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// parseIPList parses a comma-separated list of IP addresses into a map.
func parseIPList(list string) map[string]bool {
	result := make(map[string]bool)
	if list == "" {
		return result
	}

	ips := strings.Split(list, ",")
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			result[ip] = true
		}
	}

	return result
}

