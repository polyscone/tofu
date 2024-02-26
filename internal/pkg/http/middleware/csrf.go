package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/size"
	"github.com/polyscone/tofu/internal/web/httputil"
)

const (
	CSRFTokenCookieName         = "__Host-csrf"
	CSRFTokenCookieNameInsecure = "csrf"
	CSRFTokenHeaderName         = "x-csrf-token"
	CSRFTokenFieldName          = "_csrf"
)

func CSRF(errorHandler ErrorHandler) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := CSRFTokenCookieNameInsecure
			if httputil.IsTLS(r) {
				name = CSRFTokenCookieName
			}

			var cookieToken []byte
			cookie, err := r.Cookie(name)
			if !errors.Is(err, http.ErrNoCookie) {
				if handleError(w, r, err, errorHandler, http.StatusInternalServerError) {
					return
				}
			}
			if err == nil {
				decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
				if handleError(w, r, err, errorHandler, http.StatusInternalServerError) {
					return
				}

				cookieToken = decoded
			}

			ctx := r.Context()
			ctx, err = csrf.SetToken(ctx, cookieToken)
			if handleError(w, r, err, errorHandler, http.StatusInternalServerError) {
				return
			}

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				// Do nothing for safe methods

			default:
				if r.PostForm == nil {
					err := r.ParseMultipartForm(32 * size.Megabyte)
					if err != nil && !errors.Is(err, http.ErrNotMultipart) {
						handleError(w, r, err, errorHandler, http.StatusInternalServerError)

						return
					}
				}

				sentToken := r.Header.Get(CSRFTokenHeaderName)
				if sentToken == "" {
					sentToken = r.PostFormValue(CSRFTokenFieldName)
				}
				if sentToken == "" {
					handleError(w, r, csrf.ErrEmptyToken, errorHandler, http.StatusBadRequest)

					return
				}

				decoded, err := base64.RawURLEncoding.DecodeString(sentToken)
				if handleError(w, r, err, errorHandler, http.StatusInternalServerError) {
					return
				}

				err = csrf.Check(ctx, decoded)
				if handleError(w, r, err, errorHandler, http.StatusInternalServerError) {
					return
				}
			}

			found := slices.ContainsFunc(w.Header().Values("vary"), func(el string) bool {
				return strings.ToLower(el) == "cookie"
			})
			if !found {
				w.Header().Add("vary", "cookie")
			}

			rw := &csrfResponseWriter{
				ResponseWriter: w,
				r:              r,
				ctx:            ctx,
				insecure:       !httputil.IsTLS(r),
			}
			r = r.WithContext(ctx)

			next(rw, r)

			rw.commit()
		}
	}
}

var _ Unwrapper = (*csrfResponseWriter)(nil)

type csrfResponseWriter struct {
	http.ResponseWriter
	mu        sync.Mutex
	r         *http.Request
	ctx       context.Context
	insecure  bool
	committed bool
}

func (w *csrfResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *csrfResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.commit()

	return w.ResponseWriter.Write(b)
}

func (w *csrfResponseWriter) WriteHeader(statusCode int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.commit()

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *csrfResponseWriter) commit() {
	if w.committed {
		return
	}

	w.committed = true

	if csrf.IsNew(w.ctx) {
		name := CSRFTokenCookieName
		if w.insecure {
			name = CSRFTokenCookieNameInsecure
		}

		encoded := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(w.ctx))

		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    encoded,
			Path:     "/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   !w.insecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}
