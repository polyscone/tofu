package middleware

import "net/http"

func MaxBytes(maxBytes func(*http.Request) int) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes(r)))

			next(w, r)
		}
	}
}
