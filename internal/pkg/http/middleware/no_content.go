package middleware

import (
	"fmt"
	"net/http"
	"sync"
)

func NoContent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &noContentResponseWriter{ResponseWriter: w}

		next(rw, r)

		if !rw.header && !rw.body {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

var _ Unwrapper = (*noContentResponseWriter)(nil)

type noContentResponseWriter struct {
	http.ResponseWriter
	mu     sync.Mutex
	header bool
	body   bool
}

func (w *noContentResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *noContentResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

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
	w.mu.Lock()
	defer w.mu.Unlock()

	w.header = true

	w.ResponseWriter.WriteHeader(statusCode)
}
