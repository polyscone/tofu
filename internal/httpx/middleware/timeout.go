package middleware

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

type TimeoutConfig struct {
	Logger       func(r *http.Request) *slog.Logger
	ErrorHandler ErrorHandler
}

// Timeout returns a new timeout middleware configured using the given TTL.
//
// If any response is written/flushed, or if the request is hijacked then
// the timeout is ignored.
//
// If the write deadline is set through an http.ResponseController then
// the timeout will be extended to just before that time if the original timeout
// would expire before then.
// Any error handlers should extend the write deadline if needed.
//
// Any writes to the handler's ResponseWriter after the deadline will return
// an http.ErrHandlerTimeout if the timeout has not been ignored due to an
// earlier write before the deadline.
//
// On timeout the configured error handler will be called to allow for a custom
// response, or a default gateway timeout response will be sent.
func Timeout(ttl time.Duration, config *TimeoutConfig) Middleware {
	if config == nil {
		config = &TimeoutConfig{}
	}
	if config.Logger == nil {
		config.Logger = func(r *http.Request) *slog.Logger {
			return slog.Default()
		}
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			config.Logger(r).Error("timeout middleware", "error", err)

			http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			r = r.WithContext(ctx)

			rw := &timeoutWriter{
				ResponseWriter: w,
				rc:             http.NewResponseController(w),
				h:              w.Header().Clone(),
				config:         config,
			}

			done := make(chan struct{})
			_panic := make(chan any, 1)

			go func() {
				defer func() {
					if p := recover(); p != nil {
						_panic <- fmt.Sprintf("%v\n\n%s", p, debug.Stack())
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
				// Do nothing; handler executed

			case <-timeout.C:
				rw.mu.Lock()

				if rw.written {
					rw.mu.Unlock()

					goto TimeoutSelect
				}

				// If we observed that the write deadline was set to some other
				// value then we reset the timeout to a duration that would end
				// just before the new write deadline
				//
				// The new timeout duration ends just before the write deadline to
				// give error handlers a chance to extend it if they want to write
				// anything in the response writer
				const spill = 10 * time.Millisecond
				if d := time.Until(rw.deadline) - spill; !rw.deadline.IsZero() && d > 0 {
					timeout.Reset(d)

					rw.mu.Unlock()

					goto TimeoutSelect
				}

				rw.err = http.ErrHandlerTimeout

				cancel()

				// Don't try to use the response writer if it's already been written to
				// or if the connection has been hijacked
				if rw.written {
					config.Logger(r).Error("timeout middleware", "error", rw.err)
				} else {
					config.ErrorHandler(w, r, rw.err)
				}

				rw.mu.Unlock()

			case <-ctx.Done():
				rw.mu.Lock()

				if rw.written {
					rw.mu.Unlock()

					goto TimeoutSelect
				}

				switch err := ctx.Err(); err {
				case context.DeadlineExceeded:
					rw.err = http.ErrHandlerTimeout

				default:
					rw.err = err
				}

				// Don't try to use the response writer if it's already been written to
				// or if the connection has been hijacked
				if rw.written {
					config.Logger(r).Error("timeout middleware", "error", rw.err)
				} else {
					config.ErrorHandler(w, r, rw.err)
				}

				rw.mu.Unlock()
			}
		}
	}
}

var _ Unwrapper = (*timeoutWriter)(nil)

type timeoutWriter struct {
	http.ResponseWriter
	mu       sync.Mutex
	err      error
	rc       *http.ResponseController
	h        http.Header
	config   *TimeoutConfig
	written  bool
	deadline time.Time
}

func (w *timeoutWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *timeoutWriter) SetWriteDeadline(deadline time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return w.err
	}

	w.deadline = deadline

	return w.rc.SetWriteDeadline(deadline)
}

func (w *timeoutWriter) FlushError() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return w.err
	}

	w.written = true

	w.copyHeaders()

	return w.rc.Flush()
}

func (w *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return nil, nil, w.err
	}

	conn, bufrw, err := w.rc.Hijack()
	if err == nil {
		w.written = true
	}

	return conn, bufrw, err
}

func (w *timeoutWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return 0, w.err
	}

	w.written = true

	w.copyHeaders()

	return w.ResponseWriter.Write(b)
}

func (w *timeoutWriter) WriteHeader(statusCode int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return
	}

	w.written = true

	w.copyHeaders()

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *timeoutWriter) Header() http.Header {
	return w.h
}

func (w *timeoutWriter) copyHeaders() {
	// Since the main handler function is run in another goroutine
	// it means the header map can be accessed from multiple goroutines
	// through the use of w.Header(), which can cause data races
	//
	// To prevent potential data races and prevent triggering the race detector
	// the handler needs its own header map which we have to key-wise copy
	// into the actual response writer header map
	dst := w.ResponseWriter.Header()
	for key, value := range w.h {
		dst[key] = value
	}
}
