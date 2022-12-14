package middleware

import "net/http"

func Heartbeat(endpoint string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.URL.Path == endpoint {
				w.Header().Set("content-type", "application/json")
				w.Header().Set("cache-control", "no-cache")

				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusOK)

					w.Write([]byte(`{"status":"available"}`))
				}

				return
			}

			next(w, r)
		}
	}
}
