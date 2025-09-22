package safety

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	capacity     int           // Maximum number of tokens
	tokens       int           // Current number of tokens
	refillRate   int           // Tokens added per second
	lastRefill   time.Time     // Last time tokens were added
	mutex        sync.Mutex    // Protects token count
	name         string        // Name for logging/identification
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(name string, capacity, refillRate int) *RateLimiter {
	return &RateLimiter{
		capacity:   capacity,
		tokens:     capacity, // Start with full capacity
		refillRate: refillRate,
		lastRefill: time.Now(),
		name:       name,
	}
}

// Allow checks if an operation is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	return rl.AllowN(1)
}

// AllowN checks if N operations are allowed under the rate limit
func (rl *RateLimiter) AllowN(n int) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.refillTokens()

	if rl.tokens >= n {
		rl.tokens -= n
		return true
	}

	return false
}

// Wait waits until an operation is allowed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.WaitN(ctx, 1)
}

// WaitN waits until N operations are allowed
func (rl *RateLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if rl.AllowN(n) {
			return nil
		}

		// Calculate how long to wait for next token
		waitTime := rl.calculateWaitTime(n)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to check again
		}
	}
}

// refillTokens adds tokens based on elapsed time
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	if elapsed < time.Second {
		return // Not enough time has passed
	}

	// Calculate tokens to add
	tokensToAdd := int(elapsed.Seconds()) * rl.refillRate
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.capacity {
			rl.tokens = rl.capacity
		}
		rl.lastRefill = now
	}
}

// calculateWaitTime calculates how long to wait for N tokens
func (rl *RateLimiter) calculateWaitTime(n int) time.Duration {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.refillTokens()

	if rl.tokens >= n {
		return 0
	}

	tokensNeeded := n - rl.tokens
	secondsToWait := float64(tokensNeeded) / float64(rl.refillRate)
	
	// Add small buffer to account for timing precision
	return time.Duration(secondsToWait*1000+100) * time.Millisecond
}

// GetStats returns current statistics about the rate limiter
func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.refillTokens()

	return RateLimiterStats{
		Name:       rl.name,
		Capacity:   rl.capacity,
		Tokens:     rl.tokens,
		RefillRate: rl.refillRate,
		LastRefill: rl.lastRefill,
	}
}

// RateLimiterStats holds statistics about a rate limiter
type RateLimiterStats struct {
	Name       string
	Capacity   int
	Tokens     int
	RefillRate int
	LastRefill time.Time
}

// RateLimiterManager manages multiple rate limiters
type RateLimiterManager struct {
	limiters map[string]*RateLimiter
	mutex    sync.RWMutex
}

// NewRateLimiterManager creates a new rate limiter manager
func NewRateLimiterManager() *RateLimiterManager {
	return &RateLimiterManager{
		limiters: make(map[string]*RateLimiter),
	}
}

// GetOrCreate gets an existing rate limiter or creates a new one
func (rlm *RateLimiterManager) GetOrCreate(name string, capacity, refillRate int) *RateLimiter {
	rlm.mutex.RLock()
	if rl, exists := rlm.limiters[name]; exists {
		rlm.mutex.RUnlock()
		return rl
	}
	rlm.mutex.RUnlock()

	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()

	// Double-check after acquiring write lock
	if rl, exists := rlm.limiters[name]; exists {
		return rl
	}

	rl := NewRateLimiter(name, capacity, refillRate)
	rlm.limiters[name] = rl
	return rl
}

// Get gets an existing rate limiter
func (rlm *RateLimiterManager) Get(name string) (*RateLimiter, bool) {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()

	rl, exists := rlm.limiters[name]
	return rl, exists
}

// GetStats returns statistics for all rate limiters
func (rlm *RateLimiterManager) GetStats() []RateLimiterStats {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()

	stats := make([]RateLimiterStats, 0, len(rlm.limiters))
	for _, rl := range rlm.limiters {
		stats = append(stats, rl.GetStats())
	}
	return stats
}

// AdaptiveRateLimiter adjusts its rate based on success/failure patterns
type AdaptiveRateLimiter struct {
	baseLimiter    *RateLimiter
	successCount   int
	failureCount   int
	lastAdjustment time.Time
	adjustInterval time.Duration
	minRate        int
	maxRate        int
	currentRate    int
	mutex          sync.Mutex
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(name string, capacity, initialRate, minRate, maxRate int) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimiter:    NewRateLimiter(name, capacity, initialRate),
		adjustInterval: 30 * time.Second,
		minRate:        minRate,
		maxRate:        maxRate,
		currentRate:    initialRate,
		lastAdjustment: time.Now(),
	}
}

// Allow checks if an operation is allowed and records the attempt
func (arl *AdaptiveRateLimiter) Allow() bool {
	return arl.baseLimiter.Allow()
}

// RecordSuccess records a successful operation
func (arl *AdaptiveRateLimiter) RecordSuccess() {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()

	arl.successCount++
	arl.adjustRateIfNeeded()
}

// RecordFailure records a failed operation
func (arl *AdaptiveRateLimiter) RecordFailure() {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()

	arl.failureCount++
	arl.adjustRateIfNeeded()
}

// adjustRateIfNeeded adjusts the rate based on success/failure ratio
func (arl *AdaptiveRateLimiter) adjustRateIfNeeded() {
	if time.Since(arl.lastAdjustment) < arl.adjustInterval {
		return
	}

	totalOperations := arl.successCount + arl.failureCount
	if totalOperations < 10 {
		return // Not enough data
	}

	successRate := float64(arl.successCount) / float64(totalOperations)
	
	var newRate int
	if successRate > 0.95 {
		// High success rate, increase rate
		newRate = min(arl.currentRate+1, arl.maxRate)
	} else if successRate < 0.8 {
		// Low success rate, decrease rate
		newRate = max(arl.currentRate-1, arl.minRate)
	} else {
		// Acceptable success rate, maintain current rate
		newRate = arl.currentRate
	}

	if newRate != arl.currentRate {
		arl.currentRate = newRate
		arl.baseLimiter = NewRateLimiter(arl.baseLimiter.name, arl.baseLimiter.capacity, newRate)
	}

	// Reset counters
	arl.successCount = 0
	arl.failureCount = 0
	arl.lastAdjustment = time.Now()
}

// GetCurrentRate returns the current rate
func (arl *AdaptiveRateLimiter) GetCurrentRate() int {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()
	return arl.currentRate
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BurstRateLimiter allows occasional bursts while maintaining overall rate
type BurstRateLimiter struct {
	normalLimiter *RateLimiter
	burstLimiter  *RateLimiter
	burstWindow   time.Duration
	lastBurst     time.Time
	mutex         sync.Mutex
}

// NewBurstRateLimiter creates a new burst rate limiter
func NewBurstRateLimiter(name string, normalCapacity, normalRate, burstCapacity, burstRate int, burstWindow time.Duration) *BurstRateLimiter {
	return &BurstRateLimiter{
		normalLimiter: NewRateLimiter(name+"_normal", normalCapacity, normalRate),
		burstLimiter:  NewRateLimiter(name+"_burst", burstCapacity, burstRate),
		burstWindow:   burstWindow,
	}
}

// Allow checks if an operation is allowed, trying burst first then normal
func (brl *BurstRateLimiter) Allow() bool {
	brl.mutex.Lock()
	defer brl.mutex.Unlock()

	// Try burst limiter first if we're within burst window
	if time.Since(brl.lastBurst) < brl.burstWindow && brl.burstLimiter.Allow() {
		return true
	}

	// Try normal limiter
	if brl.normalLimiter.Allow() {
		brl.lastBurst = time.Now()
		return true
	}

	return false
}

// GetStats returns statistics for the burst rate limiter
func (brl *BurstRateLimiter) GetStats() (normal, burst RateLimiterStats) {
	return brl.normalLimiter.GetStats(), brl.burstLimiter.GetStats()
}
