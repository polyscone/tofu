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

var defaultTimeoutConfig = TimeoutConfig{
	ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
	},
	Logger: func(r *http.Request) *slog.Logger {
		return slog.Default()
	},
}

// Timeout returns a new timeout middleware configured using the given TTL.
//
// If a response is flushed at all then the timeout is ignored and left up to
// the handler that called Flush().
//
// If the request's write deadline is set through an http.ResponseController then
// the timeout will be extended to at least that time if the original timeout
// expires before then.
func Timeout(ttl time.Duration, config *TimeoutConfig) Middleware {
	if config == nil {
		config = &defaultTimeoutConfig
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultTimeoutConfig.ErrorHandler
	}
	if config.Logger == nil {
		config.Logger = defaultTimeoutConfig.Logger
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			r = r.WithContext(ctx)

			rw := &timeoutWriter{
				ResponseWriter: w,
				h:              make(http.Header),
				r:              r,
				rc:             http.NewResponseController(w),
				config:         config,
			}

			done := make(chan struct{})
			_panic := make(chan any, 1)

			go func() {
				defer func() {
					if p := recover(); p != nil {
						_panic <- fmt.Sprintf("%v\npreserved stack trace:\n%s", p, debug.Stack())
					}
				}()

				next(rw, r)

				close(done)
			}()

			timeout := time.NewTimer(ttl)

		TimeoutSelect:
			select {
			case p := <-_panic:
				panic(p)

			case <-done:
				rw.mu.Lock()
				defer rw.mu.Unlock()

				dst := w.Header()
				for key, value := range rw.h {
					dst[key] = value
				}

				if !rw.flushed && rw.statusCode != 0 {
					w.WriteHeader(rw.statusCode)
				}

				w.Write(rw.buf.Bytes())

			case <-timeout.C:
				rw.mu.Lock()

				// If we already flushed data to the client we ignore the timeout
				// and let whatever handler is below us decide what to do
				if rw.flushed {
					rw.mu.Unlock()

					goto TimeoutSelect
				}

				// If the write timeout was set to some other value then we
				// reset the timeout to a duration that would end just after the
				// new write timeout
				if rw.deadline.After(time.Now()) {
					const spill = 500 * time.Millisecond

					timeout.Reset(time.Until(rw.deadline) + spill)

					rw.mu.Unlock()

					goto TimeoutSelect
				}

				rw.mu.Unlock()

				cancel()

				config.ErrorHandler(w, r, http.ErrHandlerTimeout)

			case <-ctx.Done():
				rw.mu.Lock()

				// If we already flushed data to the client we ignore the context
				// and let whatever handler is below us decide what to do
				if rw.flushed {
					rw.mu.Unlock()

					goto TimeoutSelect
				}

				rw.mu.Unlock()

				switch err := ctx.Err(); err {
				case context.DeadlineExceeded:
					config.ErrorHandler(w, r, http.ErrHandlerTimeout)

				default:
					config.ErrorHandler(w, r, err)
				}
			}
		}
	}
}

var _ Unwrapper = (*timeoutWriter)(nil)

type timeoutWriter struct {
	http.ResponseWriter
	mu         sync.Mutex
	h          http.Header
	r          *http.Request
	rc         *http.ResponseController
	buf        bytes.Buffer
	config     *TimeoutConfig
	flushed    bool
	deadline   time.Time
	statusCode int
}

func (w *timeoutWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *timeoutWriter) SetWriteDeadline(deadline time.Time) error {
	w.mu.Lock()

	if deadline.After(w.deadline) {
		w.deadline = deadline
	}

	w.mu.Unlock()

	return w.rc.SetWriteDeadline(deadline)
}

func (w *timeoutWriter) FlushError() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.flushed {
		dst := w.ResponseWriter.Header()
		for key, value := range w.h {
			dst[key] = value
		}

		if w.statusCode == 0 {
			w.statusCode = http.StatusOK
		}

		w.ResponseWriter.WriteHeader(w.statusCode)
	}

	w.flushed = true

	if _, err := w.buf.WriteTo(w.ResponseWriter); err != nil {
		return fmt.Errorf("timeout: flush response buffer: %w", err)
	}

	return w.rc.Flush()
}

func (w *timeoutWriter) Header() http.Header {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.h
}

func (w *timeoutWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.Write(b)
}

func (w *timeoutWriter) WriteHeader(statusCode int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.statusCode == 0 {
		w.statusCode = statusCode
	} else {
		w.config.Logger(w.r).Error("timeout: superfluous response.WriteHeader call")
	}
}