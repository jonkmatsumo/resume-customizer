package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	// Test basic allow/deny logic
	bucket := newTokenBucket(10, 1.0) // 10 tokens, 1 token per second

	// Should allow 10 requests immediately (burst)
	for i := 0; i < 10; i++ {
		if !bucket.allow() {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// 11th request should be denied (no tokens left)
	if bucket.allow() {
		t.Error("Expected 11th request to be denied")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket := newTokenBucket(10, 1.0) // 1 token per second

	// Consume all tokens
	for i := 0; i < 10; i++ {
		bucket.allow()
	}

	// Wait for 1 token to refill
	time.Sleep(1100 * time.Millisecond)

	// Should allow one more request
	if !bucket.allow() {
		t.Error("Expected request to be allowed after refill")
	}

	// Should be denied again
	if bucket.allow() {
		t.Error("Expected request to be denied after consuming refilled token")
	}
}

func TestTokenBucket_GetStatus(t *testing.T) {
	bucket := newTokenBucket(10, 1.0)

	// Consume 5 tokens
	for i := 0; i < 5; i++ {
		bucket.allow()
	}

	remaining, resetTime := bucket.getStatus()
	if remaining != 5 {
		t.Errorf("Expected 5 remaining tokens, got %d", remaining)
	}

	if resetTime.Before(time.Now()) {
		t.Error("Reset time should be in the future")
	}
}

func TestLimiter_Allow(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultLimit:  10,
		DefaultWindow: time.Minute,
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	clientID := "127.0.0.1"
	endpoint := "/test"
	method := "GET"

	// Should allow requests up to limit
	for i := 0; i < 10; i++ {
		allowed, rateInfo := limiter.Allow(clientID, endpoint, method)
		if !allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
		if rateInfo.Limit != 10 {
			t.Errorf("Expected limit 10, got %d", rateInfo.Limit)
		}
		if rateInfo.Remaining != 9-i {
			t.Errorf("Expected remaining %d, got %d", 9-i, rateInfo.Remaining)
		}
	}

	// 11th request should be denied
	allowed, rateInfo := limiter.Allow(clientID, endpoint, method)
	if allowed {
		t.Error("Expected 11th request to be denied")
	}
	if rateInfo.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", rateInfo.Remaining)
	}
	if rateInfo.RetryAfter <= 0 {
		t.Error("Expected retry after to be positive")
	}
}

func TestLimiter_Whitelist(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultLimit:  1,
		DefaultWindow: time.Minute,
		Whitelist:     map[string]bool{"127.0.0.1": true},
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	// Whitelisted IP should always be allowed
	for i := 0; i < 100; i++ {
		allowed, rateInfo := limiter.Allow("127.0.0.1", "/test", "GET")
		if !allowed {
			t.Errorf("Expected whitelisted request %d to be allowed", i+1)
		}
		if rateInfo.Limit != 0 {
			t.Errorf("Expected limit 0 for whitelisted, got %d", rateInfo.Limit)
		}
	}
}

func TestLimiter_Blacklist(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultLimit:  1000,
		DefaultWindow: time.Minute,
		Blacklist:     map[string]bool{"192.168.1.1": true},
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	// Blacklisted IP should always be denied
	allowed, _ := limiter.Allow("192.168.1.1", "/test", "GET")
	if allowed {
		t.Error("Expected blacklisted request to be denied")
	}
}

func TestLimiter_Disabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	// When disabled, all requests should be allowed
	for i := 0; i < 100; i++ {
		allowed, rateInfo := limiter.Allow("127.0.0.1", "/test", "GET")
		if !allowed {
			t.Errorf("Expected request %d to be allowed when disabled", i+1)
		}
		if rateInfo.Limit != 0 {
			t.Errorf("Expected limit 0 when disabled, got %d", rateInfo.Limit)
		}
	}
}

func TestLimiter_EndpointSpecific(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultLimit:  1000,
		DefaultWindow: time.Minute,
		EndpointConfigs: []EndpointConfig{
			{Path: "/run", Method: "POST", Limit: 5, Window: time.Hour, Burst: 5},
		},
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	clientID := "127.0.0.1"

	// Test endpoint-specific limit (burst allows 5 immediately)
	for i := 0; i < 5; i++ {
		allowed, rateInfo := limiter.Allow(clientID, "/run", "POST")
		if !allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
		if rateInfo.Limit != 5 {
			t.Errorf("Expected limit 5, got %d", rateInfo.Limit)
		}
	}

	// 6th request should be denied (limit reached)
	allowed, rateInfo := limiter.Allow(clientID, "/run", "POST")
	if allowed {
		t.Error("Expected 6th request to be denied")
	}
	if rateInfo.Limit != 5 {
		t.Errorf("Expected limit 5, got %d", rateInfo.Limit)
	}

	// Different endpoint should use default limit
	allowed, rateInfo = limiter.Allow(clientID, "/other", "GET")
	if !allowed {
		t.Error("Expected different endpoint to be allowed")
	}
	if rateInfo.Limit != 1000 {
		t.Errorf("Expected default limit 1000, got %d", rateInfo.Limit)
	}
}

func TestLimiter_Concurrent(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultLimit:  100,
		DefaultWindow: time.Minute,
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	clientID := "127.0.0.1"
	endpoint := "/test"
	method := "GET"

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Make 200 concurrent requests (should only allow 100)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _ := limiter.Allow(clientID, endpoint, method)
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should have allowed exactly 100 requests
	if allowedCount != 100 {
		t.Errorf("Expected 100 allowed requests, got %d", allowedCount)
	}
}

func TestLimiter_Cleanup(t *testing.T) {
	config := &Config{
		Enabled:         true,
		DefaultLimit:    10,
		DefaultWindow:   time.Minute,
		CleanupInterval: 100 * time.Millisecond,
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	// Create buckets for multiple clients
	for i := 0; i < 10; i++ {
		clientID := fmt.Sprintf("127.0.0.%d", i+1)
		allowed, _ := limiter.Allow(clientID, "/test", "GET")
		if !allowed {
			t.Errorf("Expected request from %s to be allowed", clientID)
		}
	}

	// Wait for cleanup
	time.Sleep(150 * time.Millisecond)

	// Access buckets to update last access time
	for i := 0; i < 5; i++ {
		clientID := fmt.Sprintf("127.0.0.%d", i+1)
		allowed, _ := limiter.Allow(clientID, "/test", "GET")
		if !allowed {
			t.Errorf("Expected request from %s to be allowed", clientID)
		}
	}

	// Wait for cleanup again
	time.Sleep(150 * time.Millisecond)

	// Buckets should still exist (we accessed them recently)
	// This is a basic test - full cleanup testing would require more time
	// Verify that accessed buckets still work
	for i := 0; i < 5; i++ {
		clientID := fmt.Sprintf("127.0.0.%d", i+1)
		allowed, _ := limiter.Allow(clientID, "/test", "GET")
		if !allowed {
			t.Errorf("Expected request from %s to still be allowed after cleanup", clientID)
		}
	}
}

func TestLimiter_Burst(t *testing.T) {
	config := &Config{
		Enabled:       true,
		DefaultLimit:  10,
		DefaultWindow: time.Minute,
		EndpointConfigs: []EndpointConfig{
			{Path: "/burst", Method: "POST", Limit: 10, Window: time.Minute, Burst: 5},
		},
	}
	limiter := NewLimiter(config)
	defer limiter.Stop()

	clientID := "127.0.0.1"

	// Should allow burst of 5 requests immediately
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.Allow(clientID, "/burst", "POST")
		if !allowed {
			t.Errorf("Expected burst request %d to be allowed", i+1)
		}
	}

	// 6th request should be denied (burst exhausted, no refill yet)
	allowed, _ := limiter.Allow(clientID, "/burst", "POST")
	if allowed {
		t.Error("Expected request after burst to be denied")
	}
}

func TestNewLimiter_NilConfig(t *testing.T) {
	limiter := NewLimiter(nil)
	defer limiter.Stop()

	if limiter == nil {
		t.Error("Expected limiter to be created with nil config")
	}

	// Should use defaults
	allowed, rateInfo := limiter.Allow("127.0.0.1", "/test", "GET")
	if !allowed {
		t.Error("Expected request to be allowed with default config")
	}
	if rateInfo.Limit != 1000 {
		t.Errorf("Expected default limit 1000, got %d", rateInfo.Limit)
	}
}

