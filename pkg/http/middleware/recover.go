package middleware

import (
	"fmt"
	"net/http"
	"runtime"
)

func Recover(errorHandler ErrorHandler) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil && err != http.ErrAbortHandler {
					w.Header().Set("connection", "close")

					const size = 64 << 10

					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]

					if errorHandler != nil {
						message := fmt.Errorf("panic serving %v: %v\n%s", r.RemoteAddr, err, buf)

						errorHandler(w, r, message)
					} else {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					}
				}
			}()

			next(w, r)
		}
	}
}
