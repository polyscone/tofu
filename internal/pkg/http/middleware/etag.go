package middleware

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/logger"
)

type etagResponseWriter struct {
	http.ResponseWriter
	w      io.Writer
	header bool
}

func (w *etagResponseWriter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *etagResponseWriter) WriteHeader(statusCode int) {
	w.header = true

	w.ResponseWriter.WriteHeader(statusCode)
}

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
			etag := hex.EncodeToString(hash.Sum(nil))

			w.Header().Set("etag", etag)

			if !rw.header && r.Header.Get("if-none-match") == etag {
				w.WriteHeader(http.StatusNotModified)

				return
			}

			if _, err := buf.WriteTo(w); err != nil {
				logger.PrintError(err)
			}
		}
	}
}
