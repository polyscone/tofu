package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type SecurityHeadersConfig struct {
	Logger func(r *http.Request) *slog.Logger
}

func SecurityHeaders(config *SecurityHeadersConfig) Middleware {
	if config == nil {
		config = &SecurityHeadersConfig{}
	}
	if config.Logger == nil {
		config.Logger = func(r *http.Request) *slog.Logger {
			return slog.Default()
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rw := &securityHeadersResponseWriter{ResponseWriter: w}

			rw.Header().Set("referrer-policy", "strict-origin-when-cross-origin")
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
				config.Logger(r).Error("security headers middleware", "error", strings.Join(messages, "\n"))
			}
		}
	}
}

var _ Unwrapper = (*securityHeadersResponseWriter)(nil)

type securityHeadersResponseWriter struct {
	http.ResponseWriter
	body bool
}

func (w *securityHeadersResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *securityHeadersResponseWriter) Write(b []byte) (int, error) {
	w.body = true

	return w.ResponseWriter.Write(b)
}
