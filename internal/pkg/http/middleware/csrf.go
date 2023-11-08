package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/size"
)

const (
	CSRFTokenCookieName         = "__Host-csrf"
	CSRFTokenCookieNameInsecure = "csrf"
	CSRFTokenHeaderName         = "x-csrf-token"
	CSRFTokenFieldName          = "_csrf"
)

type CSRFConfig struct {
	Insecure     bool
	ErrorHandler ErrorHandler
}

func CSRF(config *CSRFConfig) Middleware {
	if config == nil {
		config = &CSRFConfig{}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := CSRFTokenCookieName
			if config.Insecure {
				name = CSRFTokenCookieNameInsecure
			}

			var cookieToken []byte
			cookie, err := r.Cookie(name)
			if !errors.Is(err, http.ErrNoCookie) {
				if handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError) {
					return
				}
			}
			if err == nil {
				decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
				if handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError) {
					return
				}

				cookieToken = decoded
			}

			ctx := r.Context()
			ctx, err = csrf.SetToken(ctx, cookieToken)
			if handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError) {
				return
			}

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				// Do nothing for safe methods

			default:
				if r.PostForm == nil {
					err := r.ParseMultipartForm(32 * size.Megabyte)
					if err != nil && !errors.Is(err, http.ErrNotMultipart) {
						handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError)

						return
					}
				}

				sentToken := r.Header.Get(CSRFTokenHeaderName)
				if sentToken == "" {
					sentToken = r.PostFormValue(CSRFTokenFieldName)
				}
				if sentToken == "" {
					handleError(w, r, csrf.ErrEmptyToken, config.ErrorHandler, http.StatusBadRequest)

					return
				}

				decoded, err := base64.RawURLEncoding.DecodeString(sentToken)
				if handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError) {
					return
				}

				err = csrf.Check(ctx, decoded)
				if handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError) {
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
				config:         config,
				ctx:            ctx,
				insecure:       config.Insecure,
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
	r         *http.Request
	config    *CSRFConfig
	ctx       context.Context
	insecure  bool
	committed bool
}

func (w *csrfResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *csrfResponseWriter) Write(b []byte) (int, error) {
	w.commit()

	return w.ResponseWriter.Write(b)
}

func (w *csrfResponseWriter) WriteHeader(statusCode int) {
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
		if w.config.Insecure {
			name = CSRFTokenCookieNameInsecure
		}

		encoded := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(w.ctx))

		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    encoded,
			Path:     "/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   !w.config.Insecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}
