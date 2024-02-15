package httputil

import "net/http"

type ServeMux interface {
	Handler(r *http.Request) (h http.Handler, pattern string)
}

var methods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

func MethodNotAllowed(mux ServeMux, r *http.Request) ([]string, bool) {
	method := r.Method
	_, current := mux.Handler(r)
	var allowed []string
	for _, method := range methods {
		// If we find a pattern that's different from the pattern for the current
		// fallback handler then we know there are actually other handlers that
		// could match with a method change, so we should handle as method not allowed
		r.Method = method
		if _, pattern := mux.Handler(r); pattern != current {
			allowed = append(allowed, method)
		}
	}

	r.Method = method

	return allowed, len(allowed) > 0
}

func RewriteHandler(handler http.Handler, path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = path

		handler.ServeHTTP(w, r)
	})
}
