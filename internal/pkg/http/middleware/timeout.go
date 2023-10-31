package middleware

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

type TimeoutConfig struct {
	ErrorHandler ErrorHandler
	Logger       func(r *http.Request) *slog.Logger
}

func Timeout(dt time.Duration, config *TimeoutConfig) Middleware {
	if config == nil {
		config = &TimeoutConfig{}
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
		}
	}
	if config.Logger == nil {
		config.Logger = func(r *http.Request) *slog.Logger {
			return slog.Default()
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelCtx := context.WithTimeout(r.Context(), dt)
			defer cancelCtx()

			r = r.WithContext(ctx)

			tw := &timeoutWriter{
				h:      make(http.Header),
				w:      w,
				r:      r,
				config: config,
			}

			doneChan := make(chan struct{})
			panicChan := make(chan any, 1)

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- fmt.Sprintf("%v\npreserved stack trace:\n%s", p, debug.Stack())
					}
				}()

				next(tw, r)

				close(doneChan)
			}()

			select {
			case p := <-panicChan:
				panic(p)

			case <-doneChan:
				tw.mu.Lock()
				defer tw.mu.Unlock()

				dst := w.Header()
				for k, vv := range tw.h {
					dst[k] = vv
				}

				if !tw.wroteHeader {
					tw.statusCode = http.StatusOK
				}

				w.WriteHeader(tw.statusCode)
				w.Write(tw.wbuf.Bytes())

			case <-ctx.Done():
				tw.mu.Lock()
				defer tw.mu.Unlock()

				switch err := ctx.Err(); err {
				case context.DeadlineExceeded:
					config.ErrorHandler(w, r, http.ErrHandlerTimeout)

					tw.err = http.ErrHandlerTimeout

				default:
					config.ErrorHandler(w, r, err)

					tw.err = err
				}
			}
		}
	}
}

var _ http.Pusher = (*timeoutWriter)(nil)

type timeoutWriter struct {
	mu          sync.Mutex
	h           http.Header
	w           http.ResponseWriter
	r           *http.Request
	wbuf        bytes.Buffer
	config      *TimeoutConfig
	err         error
	wroteHeader bool
	statusCode  int
}

func (w *timeoutWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

func (w *timeoutWriter) Header() http.Header {
	return w.h
}

func (w *timeoutWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return 0, w.err
	}

	if !w.wroteHeader {
		w.writeHeaderLocked(http.StatusOK)
	}

	return w.wbuf.Write(p)
}

func (w *timeoutWriter) writeHeaderLocked(statusCode int) {
	if statusCode < 100 || statusCode > 999 {
		panic(fmt.Sprintf("timeout writer: invalid WriteHeader code %v", statusCode))
	}

	switch {
	case w.err != nil:
		return

	case w.wroteHeader:
		if w.r != nil {
			w.config.Logger(w.r).Error("timeout writer: superfluous response.WriteHeader call")
		}

	default:
		w.wroteHeader = true
		w.statusCode = statusCode
	}
}

func (w *timeoutWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.writeHeaderLocked(code)
}
