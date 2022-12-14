package rate

import (
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var ErrInsufficientTokens = errors.New("insufficient tokens")

// TokenBucket implements a simple leaky bucket rate limiter.
// It is safe to use concurrently, though it should be noted that some token
// loss may occur around the limits of the bucket's capacity.
type TokenBucket struct {
	mu            sync.Mutex
	capacity      float64
	replenish     float64
	tokens        float64
	replenishedAt time.Time
}

// NewTokenBucket returns a new leaky token bucket where the number of tokens
// is equal to the capacity.
//
// The capacity represents the maximum number of tokens in the bucket, and the
// replenish parameter represents the rate tokens are replenished per second.
func NewTokenBucket(capacity, replenish int) *TokenBucket {
	return &TokenBucket{
		capacity:      float64(capacity),
		replenish:     float64(replenish),
		tokens:        float64(capacity),
		replenishedAt: time.Now(),
	}
}

// Leak will first replenish r*s tokens, where r is the replenish number set at
// bucket creation, and s is the number of seconds since the last time the
// bucket was replenished.
//
// The last time the bucket was replenished is recorded as the time
// argument passed in. remove n tokens from the bucket.
//
// After replenishing tokens it will then leak n tokens.
// An error is returned if the bucket has less than n tokens before leaking.
func (tb *TokenBucket) Leak(n int, t time.Time) (int, error) {
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

	if tb.tokens < float64(n) {
		return int(tb.tokens), errors.Tracef(ErrInsufficientTokens)
	}

	if n > 0 {
		tb.tokens -= float64(n)

		if tb.tokens < 0 {
			tb.tokens = 0
		}
	}

	return int(tb.tokens), nil
}
