package httpx

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

func MethodNotAllowed(mux ServeMux, r *http.Request) (_ []string, notAllowed bool) {
	method := r.Method
	_, current := mux.Handler(r)
	var allowedMethods []string
	for _, method := range methods {
		// If we find a pattern that's different from the pattern for the current
		// fallback handler then we know there are actually other handlers that
		// could match with a method change, so we should handle as method not allowed
		r.Method = method
		if _, pattern := mux.Handler(r); pattern != current {
			allowedMethods = append(allowedMethods, method)
		}
	}

	r.Method = method

	return allowedMethods, len(allowedMethods) > 0
}

func RewriteHandler(handler http.Handler, path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = path

		handler.ServeHTTP(w, r)
	})
}

func IsTLS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("x-forwarded-proto") == "https"
}
