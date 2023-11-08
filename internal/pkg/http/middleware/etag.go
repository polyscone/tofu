package middleware

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
)

type ETagConfig struct {
	Logger func(r *http.Request) *slog.Logger
}

func ETag(config *ETagConfig) Middleware {
	if config == nil {
		config = &ETagConfig{}
	}
	if config.Logger == nil {
		config.Logger = func(r *http.Request) *slog.Logger {
			return slog.Default()
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next(w, r)

				return
			}

			var buf bytes.Buffer
			hash := md5.New()

			rw := &etagResponseWriter{
				ResponseWriter: w,
				w:              io.MultiWriter(&buf, hash),
				r:              r,
				config:         config,
			}

			next(rw, r)

			if buf.Len() > 0 {
				var etag string
				if etags := w.Header().Values("etag"); len(etags) != 0 {
					etag = etags[0]

					w.Header().Set("etag", etag)
				} else {
					etag = hex.EncodeToString(hash.Sum(nil))

					w.Header().Set("etag", etag)
				}

				if r.Header.Get("if-none-match") == etag {
					w.WriteHeader(http.StatusNotModified)

					return
				}

				if rw.statusCode != 0 {
					w.WriteHeader(rw.statusCode)
				}

				if _, err := buf.WriteTo(w); err != nil {
					config.Logger(r).Error("etag: write response", "error", err)
				}
			}
		}
	}
}

var _ Unwrapper = (*etagResponseWriter)(nil)

type etagResponseWriter struct {
	http.ResponseWriter
	w          io.Writer
	r          *http.Request
	config     *ETagConfig
	statusCode int
}

func (w *etagResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *etagResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

func (w *etagResponseWriter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *etagResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	} else {
		w.config.Logger(w.r).Error("timeout writer: superfluous response.WriteHeader call")
	}
}
