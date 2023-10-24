package middleware

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
)

func ETag(next http.HandlerFunc) http.HandlerFunc {
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
				slog.Error("etag: write response", "error", err)
			}
		}
	}
}

var _ http.Pusher = (*etagResponseWriter)(nil)

type etagResponseWriter struct {
	http.ResponseWriter
	w          io.Writer
	statusCode int
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
		slog.Error("timeout writer: superfluous response.WriteHeader call")
	}
}
