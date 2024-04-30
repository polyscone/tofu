package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

func NoContent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &noContentResponseWriter{
			ResponseWriter: w,
			rc:             http.NewResponseController(w),
		}

		next(rw, r)

		if !rw.header && !rw.body && !rw.hijacked {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

var _ Unwrapper = (*noContentResponseWriter)(nil)

type noContentResponseWriter struct {
	http.ResponseWriter
	rc       *http.ResponseController
	header   bool
	body     bool
	hijacked bool
}

func (w *noContentResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *noContentResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, bufrw, err := w.rc.Hijack()
	if err == nil {
		w.hijacked = true
	}

	return conn, bufrw, err
}

func (w *noContentResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		err = fmt.Errorf("no content: write response: %w", err)
	}
	if n > 0 {
		w.body = true
	}

	return n, err
}

func (w *noContentResponseWriter) WriteHeader(statusCode int) {
	w.header = true

	w.ResponseWriter.WriteHeader(statusCode)
}
