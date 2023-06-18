package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

func Timeout(dt time.Duration, errorHandler ErrorHandler) Middleware {
	if errorHandler == nil {
		errorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelCtx := context.WithTimeout(r.Context(), dt)
			defer cancelCtx()

			r = r.WithContext(ctx)

			tw := &timeoutWriter{
				w:   w,
				h:   make(http.Header),
				req: r,
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
					tw.code = http.StatusOK
				}

				w.WriteHeader(tw.code)
				w.Write(tw.wbuf.Bytes())

			case <-ctx.Done():
				tw.mu.Lock()
				defer tw.mu.Unlock()

				switch err := ctx.Err(); err {
				case context.DeadlineExceeded:
					errorHandler(w, r, http.ErrHandlerTimeout)

					tw.err = http.ErrHandlerTimeout

				default:
					errorHandler(w, r, err)

					tw.err = err
				}
			}
		}
	}
}

var _ http.Pusher = (*timeoutWriter)(nil)

type timeoutWriter struct {
	w    http.ResponseWriter
	h    http.Header
	wbuf bytes.Buffer
	req  *http.Request

	mu          sync.Mutex
	err         error
	wroteHeader bool
	code        int
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

func (w *timeoutWriter) writeHeaderLocked(code int) {
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("timeout writer: invalid WriteHeader code %v", code))
	}

	switch {
	case w.err != nil:
		return

	case w.wroteHeader:
		if w.req != nil {
			slog.Error("timeout writer: superfluous response.WriteHeader call")
		}

	default:
		w.wroteHeader = true
		w.code = code
	}
}

func (w *timeoutWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.writeHeaderLocked(code)
}
