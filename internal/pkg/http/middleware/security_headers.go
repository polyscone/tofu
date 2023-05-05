package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/logger"
)

func SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &securityHeadersResponseWriter{ResponseWriter: w}

		rw.Header().Set("x-content-type-options", "nosniff")
		rw.Header().Set("x-frame-options", "deny")

		next(rw, r)

		var messages []string

		if rw.body {
			header := "content-type"
			if got := rw.Header().Get(header); got == "" {
				messages = append(messages, fmt.Sprintf("response header %q for %v %v should be set; got empty string", header, r.Method, r.URL.Path))
			}
		}

		header := "x-content-type-options"
		if want, got := "nosniff", rw.Header().Get(header); want != got {
			messages = append(messages, fmt.Sprintf("response header %q for %v %v should be %q; got %q", header, r.Method, r.URL.Path, want, got))
		}

		header = "x-frame-options"
		if want, got := "deny", rw.Header().Get(header); want != got {
			messages = append(messages, fmt.Sprintf("response header %q for %v %v should be %q; got %q", header, r.Method, r.URL.Path, want, got))
		}

		if len(messages) > 0 {
			logger.Error.Println(strings.Join(messages, "\n"))
		}
	}
}

var _ http.Pusher = (*securityHeadersResponseWriter)(nil)

type securityHeadersResponseWriter struct {
	http.ResponseWriter
	body bool
}

func (w *securityHeadersResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

func (w *securityHeadersResponseWriter) Write(b []byte) (int, error) {
	w.body = true

	return w.ResponseWriter.Write(b)
}
