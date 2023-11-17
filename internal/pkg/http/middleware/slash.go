package middleware

import (
	"net/http"
	"strings"
)

func RemoveTrailingSlash(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "" && r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")

			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)

			return
		}

		next(w, r)
	}
}
