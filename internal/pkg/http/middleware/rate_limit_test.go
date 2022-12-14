package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
)

func TestRateLimit(t *testing.T) {
	t.Run("basic limits", func(t *testing.T) {
		capacity, replenish := 5, 1
		handler := middleware.RateLimit(capacity, replenish, nil)(func(w http.ResponseWriter, r *http.Request) {})

		tt := []struct {
			name       string
			ip         string
			requests   int
			wantStatus int
		}{
			{"one request", "1.2.3.1", 1, http.StatusOK},
			{"capacity limit", "1.2.3.2", capacity, http.StatusOK},
			{"too many requests", "1.2.3.3", capacity + 1, http.StatusTooManyRequests},
			{"ip with port capacity limit", "1.2.3.4:8080", capacity, http.StatusOK},
			{"same ip with different port too many requests", "1.2.3.4:8081", 1, http.StatusTooManyRequests},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				for i := 0; i < tc.requests; i++ {
					wantStatus := http.StatusOK
					if i == tc.requests-1 {
						wantStatus = tc.wantStatus
					}

					req := errors.Must(http.NewRequest(http.MethodGet, "/", nil))

					req.RemoteAddr = tc.ip

					w := httptest.NewRecorder()

					handler(w, req)

					res := w.Result()

					defer res.Body.Close()

					if want, got := wantStatus, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
				}
			})
		}
	})

	t.Run("rate by ip with x-forwarded-for and no trusted proxies", func(t *testing.T) {
		capacity, replenish := 1, 1
		handler := middleware.RateLimit(capacity, replenish, nil)(func(w http.ResponseWriter, r *http.Request) {})

		tt := []struct {
			name       string
			ip         string
			wantStatus int
		}{
			{"first request status ok", "1.2.3.4", http.StatusOK},
			{"second request status too many requests", "1.2.3.4", http.StatusTooManyRequests},
			{"third request status too many requests", "1.2.3.4", http.StatusTooManyRequests},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				req := errors.Must(http.NewRequest(http.MethodGet, "/", nil))

				req.RemoteAddr = tc.ip

				req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")
				req.Header.Add("x-forwarded-for", "3.3.3.3")

				w := httptest.NewRecorder()

				handler(w, req)

				res := w.Result()

				defer res.Body.Close()

				if want, got := tc.wantStatus, res.StatusCode; want != got {
					t.Errorf("want %v; got %v", want, got)
				}
			})
		}
	})

	t.Run("rate by ip with x-forwarded-for with trusted proxies", func(t *testing.T) {
		capacity, replenish := 1, 1
		handler := middleware.RateLimit(capacity, replenish, &middleware.RateLimitConfig{
			TrustedProxies: []string{"1.2.3.4", "1.1.1.1", "3.3.3.3"},
		})(func(w http.ResponseWriter, r *http.Request) {})

		tt := []struct {
			name       string
			ip         string
			wantStatus int
		}{
			{"1. first request status ok no trusted", "9.9.9.9", http.StatusOK},
			{"1. second request status too many requests no trusted", "9.9.9.9", http.StatusTooManyRequests},

			{"2. first request status ok with trusted", "1.2.3.4", http.StatusOK},
			{"2. second request status too many requests with trusted", "3.3.3.3", http.StatusTooManyRequests},
			{"2. third request status too many requests with different trusted", "1.1.1.1", http.StatusTooManyRequests},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				req := errors.Must(http.NewRequest(http.MethodGet, "/", nil))

				req.RemoteAddr = tc.ip

				req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")
				req.Header.Add("x-forwarded-for", "3.3.3.3")

				w := httptest.NewRecorder()

				handler(w, req)

				res := w.Result()

				defer res.Body.Close()

				if want, got := tc.wantStatus, res.StatusCode; want != got {
					t.Errorf("want %v; got %v", want, got)
				}
			})
		}
	})
}
