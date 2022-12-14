package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/rate"
	"github.com/polyscone/tofu/internal/pkg/realip"
)

type RateLimitConfig struct {
	TrustedProxies []string
	ErrorHandler   ErrorHandler
}

func RateLimit(capacity, replenish int, config *RateLimitConfig) Middleware {
	if config == nil {
		config = &RateLimitConfig{}
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
		lifespan := secondsUntilFull * 2

		for range time.Tick(lifespan) {
			func() {
				mu.Lock()
				defer mu.Unlock()

				for key, client := range clients {
					if time.Since(client.seenAt) > lifespan {
						delete(clients, key)
					}
				}
			}()
		}
	})

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip, err := realip.FromRequest(r, config.TrustedProxies...)
			if handleError(w, r, errors.Tracef(err), config.ErrorHandler, http.StatusInternalServerError) {
				return
			}

			client := getClient(ip)

			if _, err := client.bucket.Leak(1, time.Now()); err != nil {
				handleError(w, r, errors.Tracef(err), config.ErrorHandler, http.StatusTooManyRequests)

				return
			}

			next(w, r)
		}
	}
}
