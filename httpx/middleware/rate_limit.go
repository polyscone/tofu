package middleware

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/polyscone/tofu/background"
	"github.com/polyscone/tofu/httpx/realip"
	"github.com/polyscone/tofu/rate"
)

type ConsumeFunc func(r *http.Request) bool

type RateLimitConfig struct {
	Consume        ConsumeFunc
	TrustedProxies []string
	ErrorHandler   ErrorHandler
}

func RateLimit(capacity, replenish float64, config *RateLimitConfig) Middleware {
	if config == nil {
		config = &RateLimitConfig{}
	}

	type client struct {
		mu     sync.Mutex
		bucket *rate.TokenBucket
		seenAt time.Time
	}

	var clientsMu sync.RWMutex
	clients := make(map[string]*client)

	getClient := func(key string) *client {
		clientsMu.RLock()

		if c, ok := clients[key]; ok {
			clientsMu.RUnlock()

			c.mu.Lock()
			c.seenAt = time.Now()
			c.mu.Unlock()

			return c
		}

		clientsMu.RUnlock()

		clientsMu.Lock()
		defer clientsMu.Unlock()

		if c, ok := clients[key]; ok {
			c.mu.Lock()
			c.seenAt = time.Now()
			c.mu.Unlock()

			return c
		}

		c := &client{
			bucket: rate.NewTokenBucket(capacity, replenish),
			seenAt: time.Now(),
		}

		clients[key] = c

		return c
	}

	// Background goroutine to clean up expired clients
	background.Go(func() {
		secondsUntilFull := time.Duration(capacity/replenish) * time.Second
		ttl := secondsUntilFull * 2

		var expired []string
		for range time.Tick(ttl) {
			expired = expired[:0]

			clientsMu.RLock()

			for key, client := range clients {
				if time.Since(client.seenAt) > ttl {
					expired = append(expired, key)
				}
			}

			clientsMu.RUnlock()

			if len(expired) > 0 {
				clientsMu.Lock()

				for _, key := range expired {
					delete(clients, key)
				}

				clientsMu.Unlock()
			}
		}
	})

	capacityHeaderValue := strconv.FormatFloat(capacity, 'f', -1, 64)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if config.Consume == nil || config.Consume(r) {
				ip, err := realip.FromRequest(r, config.TrustedProxies)
				if err != nil {
					err = fmt.Errorf("realip from request: %w", err)

					handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError)

					return
				}

				client := getClient(ip)
				remaining, bucketTakeErr := client.bucket.Take(1, time.Now())

				fullReplenishDuration := time.Duration(math.Round((capacity-float64(remaining))/replenish)) * time.Second
				fullReplenishAt := time.Now().Add(fullReplenishDuration)

				w.Header().Set("x-ratelimit-limit", capacityHeaderValue)
				w.Header().Set("x-ratelimit-remaining", strconv.Itoa(remaining))
				w.Header().Set("x-ratelimit-reset", strconv.FormatInt(fullReplenishAt.Unix(), 10))

				if err := bucketTakeErr; err != nil {
					// If a client is hitting the rate limit we set the connection header to
					// close which will trigger the standard library's HTTP server to close
					// the connection after the response is sent
					//
					// Doing this means the client needs to go through the handshake process
					// again to make a new connection the next time, which should help to slow
					// down additional requests for clients that keep on hitting the limit
					//
					// This has no effect when headers have already been sent unless the HTTP
					// status code was of the 1xx class or the modified headers are trailers, but
					// this middleware should ideally be doing its checks before any handlers
					// would be writing to the response writer anyway
					w.Header().Set("connection", "close")

					handleError(w, r, err, config.ErrorHandler, http.StatusTooManyRequests)

					return
				}
			}

			next(w, r)
		}
	}
}
