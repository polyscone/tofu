package middleware

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
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
				rc:             http.NewResponseController(w),
				buf:            &buf,
				w:              io.MultiWriter(&buf, hash),
				r:              r,
				config:         config,
			}

			next(rw, r)

			if buf.Len() > 0 && !rw.flushed && !rw.hijacked {
				var etag string
				if etags := w.Header().Values("etag"); len(etags) != 0 {
					etag = etags[0]

					w.Header().Set("etag", etag)
				} else {
					etag = hex.EncodeToString(hash.Sum(nil))

					w.Header().Set("etag", etag)
				}

				w.Header().Set("cache-control", "no-cache")

				if r.Header.Get("if-none-match") == etag {
					w.WriteHeader(http.StatusNotModified)

					return
				}

				if rw.statusCode != 0 {
					w.WriteHeader(rw.statusCode)
				}

				if _, err := buf.WriteTo(w); err != nil {
					config.Logger(r).Error("etag: write end of response buffer to response: %w", err)
				}
			}
		}
	}
}

var _ Unwrapper = (*etagResponseWriter)(nil)

type etagResponseWriter struct {
	http.ResponseWriter
	rc         *http.ResponseController
	buf        *bytes.Buffer
	w          io.Writer
	r          *http.Request
	config     *ETagConfig
	flushed    bool
	hijacked   bool
	statusCode int
}

func (w *etagResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *etagResponseWriter) FlushError() error {
	if w.flushed || w.hijacked {
		return w.rc.Flush()
	}

	w.flushed = true

	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}

	w.ResponseWriter.WriteHeader(w.statusCode)

	if _, err := w.buf.WriteTo(w.ResponseWriter); err != nil {
		return fmt.Errorf("etag: write response buffer: %w", err)
	}

	return w.rc.Flush()
}

func (w *etagResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, bufrw, err := w.rc.Hijack()
	if err == nil {
		w.hijacked = true

		// If the connection was successfully hijacked we dump any
		// buffered output into the hijacked buffer so the caller can
		// decide what to do with it
		if _, err := w.buf.WriteTo(bufrw); err != nil {
			return conn, bufrw, fmt.Errorf("etag: write response buffer to hijacked buffer: %w", err)
		}
	}

	return conn, bufrw, err
}

func (w *etagResponseWriter) Write(b []byte) (int, error) {
	if w.flushed || w.hijacked {
		return w.ResponseWriter.Write(b)
	}

	return w.w.Write(b)
}

func (w *etagResponseWriter) WriteHeader(statusCode int) {
	if w.flushed || w.hijacked {
		w.ResponseWriter.WriteHeader(statusCode)

		return
	}

	if w.statusCode == 0 {
		w.statusCode = statusCode
	} else {
		w.config.Logger(w.r).Error("etag: superfluous response.WriteHeader call")
	}
}
