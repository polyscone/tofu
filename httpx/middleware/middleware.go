package middleware

import "net/http"

// Unwrapper is used to check that response writer wrappers implement
// the Unwrap method so they can be used by the http package's
// ResponseController to access features on the underlying original
// response writer, such as flushing, hijacking, etc.
type Unwrapper interface {
	Unwrap() http.ResponseWriter
}

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
