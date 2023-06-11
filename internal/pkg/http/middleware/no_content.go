package middleware

import (
	"fmt"
	"net/http"
)

func NoContent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &noContentResponseWriter{ResponseWriter: w}

		next(rw, r)

		if !rw.header && !rw.body {
			rw.WriteHeader(http.StatusNoContent)
		}
	}
}

var _ http.Pusher = (*noContentResponseWriter)(nil)

type noContentResponseWriter struct {
	http.ResponseWriter
	header bool
	body   bool
}

func (w *noContentResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

func (w *noContentResponseWriter) Write(b []byte) (int, error) {
	i, err := w.ResponseWriter.Write(b)
	if err != nil {
		err = fmt.Errorf("no content: write response: %w", err)
	}
	if i > 0 {
		w.body = true
	}

	return i, err
}

func (w *noContentResponseWriter) WriteHeader(statusCode int) {
	w.header = true

	w.ResponseWriter.WriteHeader(statusCode)
}
