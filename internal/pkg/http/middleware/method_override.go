package middleware

import "net/http"

func MethodOverride(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			method := r.URL.Query().Get("_method")
			if method == "" {
				method = r.Header.Get("x-http-method-override")
			}

			switch method {
			case http.MethodPut, http.MethodPatch, http.MethodDelete:
				r.Method = method
			}
		}

		next(w, r)
	}
}
