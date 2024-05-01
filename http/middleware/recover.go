package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
)

type RecoverConfig struct {
	Logger       func(r *http.Request) *slog.Logger
	ErrorHandler ErrorHandler
}

func Recover(config *RecoverConfig) Middleware {
	if config == nil {
		config = &RecoverConfig{}
	}
	if config.Logger == nil {
		config.Logger = func(r *http.Request) *slog.Logger {
			return slog.Default()
		}
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			config.Logger(r).Error("recover middleware", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rw := &recoverResponseWriter{
				ResponseWriter: w,
				rc:             http.NewResponseController(w),
			}

			defer func() {
				if err := recover(); err != nil && err != http.ErrAbortHandler {
					const size = 64 << 10

					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]

					errp := fmt.Errorf("panic serving %v: %v\n%s", r.RemoteAddr, err, buf)

					// If a response has been at least partially written, flushed, or
					// the connection has been hijacked we still want to log the panic
					// but not try to use the response writer
					if rw.written {
						config.Logger(r).Error("recover middleware", "error", errp)

						return
					}

					config.ErrorHandler(w, r, errp)
				}
			}()

			next(rw, r)
		}
	}
}

var _ Unwrapper = (*recoverResponseWriter)(nil)

type recoverResponseWriter struct {
	http.ResponseWriter
	rc      *http.ResponseController
	written bool
}

func (w *recoverResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *recoverResponseWriter) FlushError() error {
	w.written = true

	return w.rc.Flush()
}

func (w *recoverResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, bufrw, err := w.rc.Hijack()
	if err == nil {
		w.written = true
	}

	return conn, bufrw, err
}

func (w *recoverResponseWriter) Write(b []byte) (int, error) {
	w.written = true

	return w.ResponseWriter.Write(b)
}

func (w *recoverResponseWriter) WriteHeader(statusCode int) {
	w.written = true

	w.ResponseWriter.WriteHeader(statusCode)
}
