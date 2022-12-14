package middleware

import "net/http"

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

type Middleware func(next http.HandlerFunc) http.HandlerFunc

func Apply(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]

		handler = http.HandlerFunc(middleware(handler.ServeHTTP))
	}

	return handler
}

func handleError(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler, fallbackStatus int) bool {
	if err == nil {
		return false
	}

	if handler == nil {
		http.Error(w, http.StatusText(fallbackStatus), fallbackStatus)
	} else {
		handler(w, r, err)
	}

	return true
}
