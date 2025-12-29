// Package ratelimit provides rate limiting functionality using token bucket algorithm.
package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket represents a token bucket rate limiter.
// It allows a certain number of requests (tokens) per time window,
// with tokens refilling at a steady rate.
type TokenBucket struct {
	capacity   int           // Maximum tokens (burst capacity)
	refillRate float64       // Tokens per second
	tokens     float64       // Current tokens available
	lastRefill time.Time     // Last time tokens were refilled
	mu         sync.Mutex    // Mutex for thread safety
}

// newTokenBucket creates a new token bucket with the specified capacity and refill rate.
func newTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
		tokens:     float64(capacity), // Start with full bucket
		lastRefill: time.Now(),
	}
}

// allow checks if a token is available and consumes it if so.
// Returns true if request is allowed, false otherwise.
func (tb *TokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := elapsed.Seconds() * tb.refillRate

	// Add tokens, but don't exceed capacity
	tb.tokens = min(float64(tb.capacity), tb.tokens+tokensToAdd)
	tb.lastRefill = now

	// Check if we have at least one token
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// getStatus returns the current status of the bucket without consuming a token.
func (tb *TokenBucket) getStatus() (remaining int, resetTime time.Time) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := elapsed.Seconds() * tb.refillRate

	// Add tokens, but don't exceed capacity
	if tb.tokens+tokensToAdd > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	} else {
		tb.tokens += tokensToAdd
	}
	tb.lastRefill = now

	remaining = int(tb.tokens)
	// Calculate when bucket will be full again
	if tb.tokens < float64(tb.capacity) {
		tokensNeeded := float64(tb.capacity) - tb.tokens
		secondsUntilFull := tokensNeeded / tb.refillRate
		resetTime = now.Add(time.Duration(secondsUntilFull * float64(time.Second)))
	} else {
		resetTime = now
	}

	return remaining, resetTime
}

// Info contains information about rate limit status.
type Info struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetTime  time.Time
	RetryAfter time.Duration
}

// Limiter manages rate limiting for multiple clients using token buckets.
type Limiter struct {
	buckets        map[string]*TokenBucket // Client ID -> bucket
	mu             sync.RWMutex
	config         *Config
	cleanupTicker  *time.Ticker
	cleanupStop    chan struct{}
	lastAccess     map[string]time.Time // Track last access for cleanup
	accessMu       sync.RWMutex
}

// Config holds rate limiting configuration.
type Config struct {
	Enabled         bool
	DefaultLimit    int
	DefaultWindow   time.Duration
	CleanupInterval time.Duration
	Whitelist       map[string]bool
	Blacklist       map[string]bool
	EndpointConfigs []EndpointConfig
}

// NewLimiter creates a new rate limiter with the given configuration.
func NewLimiter(config *Config) *Limiter {
	if config == nil {
		config = &Config{
			Enabled:         true,
			DefaultLimit:    1000,
			DefaultWindow:   time.Minute,
			CleanupInterval: 5 * time.Minute,
			Whitelist:       make(map[string]bool),
			Blacklist:       make(map[string]bool),
		}
	}

	limiter := &Limiter{
		buckets:    make(map[string]*TokenBucket),
		config:     config,
		lastAccess: make(map[string]time.Time),
	}

	// Start cleanup goroutine if enabled
	if config.Enabled && config.CleanupInterval > 0 {
		limiter.cleanupTicker = time.NewTicker(config.CleanupInterval)
		limiter.cleanupStop = make(chan struct{})
		go limiter.cleanup()
	}

	return limiter
}

// Allow checks if a request from the given client is allowed for the specified endpoint.
// Returns true if allowed, false if rate limited, along with rate limit information.
func (l *Limiter) Allow(clientID string, endpoint string, method string) (bool, Info) {
	// Check if rate limiting is disabled
	if !l.config.Enabled {
		return true, Info{
			Allowed:   true,
			Limit:     0,
			Remaining: 0,
		}
	}

	// Check whitelist
	if l.config.Whitelist[clientID] {
		return true, Info{
			Allowed:   true,
			Limit:     0,
			Remaining: 0,
		}
	}

	// Check blacklist
	if l.config.Blacklist[clientID] {
		return false, Info{
			Allowed:   false,
			Limit:     0,
			Remaining: 0,
		}
	}

	// Find matching endpoint configuration
	endpointConfig := MatchEndpoint(endpoint, method, l.config.EndpointConfigs)
	if endpointConfig == nil {
		// Use global default
		endpointConfig = &EndpointConfig{
			Limit:  l.config.DefaultLimit,
			Window: l.config.DefaultWindow,
			Burst:  l.config.DefaultLimit, // Use limit as burst for default
		}
	}

	// Unlimited endpoint (e.g., health check)
	if endpointConfig.Limit <= 0 {
		return true, Info{
			Allowed:   true,
			Limit:     0,
			Remaining: 0,
		}
	}

	// Get or create bucket for this client+endpoint combination
	// Use clientID + endpoint + method as unique key
	bucketKey := clientID + ":" + endpoint + ":" + method
	bucket := l.getBucket(bucketKey, endpointConfig.Limit, endpointConfig.Window, endpointConfig.Burst)

	// Update last access time
	l.accessMu.Lock()
	l.lastAccess[bucketKey] = time.Now()
	l.accessMu.Unlock()

	// Check if request is allowed
	allowed := bucket.allow()
	remaining, resetTime := bucket.getStatus()

	// Calculate retry after if not allowed
	var retryAfter time.Duration
	if !allowed {
		retryAfter = time.Until(resetTime)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	return allowed, Info{
		Allowed:    allowed,
		Limit:      endpointConfig.Limit,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}
}

// getBucket gets or creates a token bucket for the given key.
func (l *Limiter) getBucket(key string, limit int, window time.Duration, burst int) *TokenBucket {
	l.mu.RLock()
	bucket, exists := l.buckets[key]
	l.mu.RUnlock()

	if exists {
		return bucket
	}

	// Create new bucket
	// Refill rate = limit / window duration in seconds
	refillRate := float64(limit) / window.Seconds()
	capacity := burst
	if capacity <= 0 {
		capacity = limit
	}

	bucket = newTokenBucket(capacity, refillRate)

	l.mu.Lock()
	// Double-check after acquiring write lock
	if existing, exists := l.buckets[key]; exists {
		l.mu.Unlock()
		return existing
	}
	l.buckets[key] = bucket
	l.mu.Unlock()

	return bucket
}

// cleanup removes old unused buckets to prevent memory leaks.
func (l *Limiter) cleanup() {
	for {
		select {
		case <-l.cleanupTicker.C:
			l.cleanupBuckets()
		case <-l.cleanupStop:
			return
		}
	}
}

// cleanupBuckets removes buckets that haven't been accessed in over an hour.
func (l *Limiter) cleanupBuckets() {
	cutoff := time.Now().Add(-1 * time.Hour)

	l.accessMu.RLock()
	keysToCheck := make([]string, 0, len(l.lastAccess))
	for key := range l.lastAccess {
		keysToCheck = append(keysToCheck, key)
	}
	l.accessMu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.accessMu.Lock()
	defer l.accessMu.Unlock()

	for _, key := range keysToCheck {
		if lastAccess, exists := l.lastAccess[key]; exists && lastAccess.Before(cutoff) {
			delete(l.buckets, key)
			delete(l.lastAccess, key)
		}
	}
}

// Stop stops the cleanup goroutine.
func (l *Limiter) Stop() {
	if l.cleanupTicker != nil {
		l.cleanupTicker.Stop()
	}
	if l.cleanupStop != nil {
		close(l.cleanupStop)
	}
}


