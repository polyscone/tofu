package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/rate"
	"github.com/polyscone/tofu/internal/pkg/realip"
)

type ConsumeFunc func(r *http.Request) bool

type RateLimitConfig struct {
	Consume        ConsumeFunc
	TrustedProxies []string
	ErrorHandler   ErrorHandler
}

var defaultRateLimitConfig RateLimitConfig

func RateLimit(capacity, replenish float64, config *RateLimitConfig) Middleware {
	if config == nil {
		config = &defaultRateLimitConfig
	}

	type client struct {
		bucket *rate.TokenBucket
		seenAt time.Time
	}

	var mu sync.Mutex
	clients := make(map[string]*client)

	getClient := func(key string) *client {
		mu.Lock()
		defer mu.Unlock()

		if _, ok := clients[key]; !ok {
			clients[key] = &client{
				bucket: rate.NewTokenBucket(capacity, replenish),
			}
		}

		c := clients[key]

		c.seenAt = time.Now()

		return c
	}

	// Background goroutine to clean up expired clients
	background.Go(func() {
		secondsUntilFull := time.Duration(capacity/replenish) * time.Second
		ttl := secondsUntilFull * 2

		for range time.Tick(ttl) {
			func() {
				mu.Lock()
				defer mu.Unlock()

				for key, client := range clients {
					if time.Since(client.seenAt) > ttl {
						delete(clients, key)
					}
				}
			}()
		}
	})

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if config.Consume == nil || config.Consume(r) {
				ip, err := realip.FromRequest(r, config.TrustedProxies)
				if handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError) {
					return
				}

				client := getClient(ip)

				if _, err := client.bucket.Take(1, time.Now()); err != nil {
					handleError(w, r, err, config.ErrorHandler, http.StatusTooManyRequests)

					return
				}
			}

			next(w, r)
		}
	}
}
