package rate

import (
	"errors"
	"sync"
	"time"
)

var ErrInsufficientTokens = errors.New("insufficient tokens")

// TokenBucket implements a simple token bucket rate limiter.
// It is safe to use concurrently, though it should be noted that some token
// loss may occur around the limits of the bucket's capacity.
type TokenBucket struct {
	mu            sync.Mutex
	capacity      float64
	replenish     float64
	tokens        float64
	replenishedAt time.Time
}

// NewTokenBucket returns a new token bucket where the number of tokens
// is equal to the capacity.
//
// The capacity represents the maximum number of tokens in the bucket, and the
// replenish parameter represents the rate tokens are replenished per second.
func NewTokenBucket(capacity, replenish float64) *TokenBucket {
	return &TokenBucket{
		capacity:      capacity,
		replenish:     replenish,
		tokens:        capacity,
		replenishedAt: time.Now(),
	}
}

// Take will first replenish r*s tokens, where r is the replenish number set at
// bucket creation, and s is the number of seconds since the last time the
// bucket was replenished.
//
// The last time the bucket was replenished is recorded as the time
// argument passed in.
//
// After replenishing tokens it will then take n tokens.
// An error is returned if the bucket has less than n tokens before taking.
//
// The number of remaining tokens returned always represents full tokens and any
// decimal value is truncated.
func (tb *TokenBucket) Take(n float64, t time.Time) (int, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if t.After(tb.replenishedAt) {
		seconds := t.Sub(tb.replenishedAt).Seconds()

		tb.tokens += tb.replenish * seconds
		tb.replenishedAt = t

		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
	}

	if tb.tokens < n {
		return int(tb.tokens), ErrInsufficientTokens
	}

	if n > 0 {
		tb.tokens -= n
	}

	return int(tb.tokens), nil
}

func (tb *TokenBucket) ReplenishedAt() time.Time {
	return tb.replenishedAt
}
