package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strings"

	"github.com/polyscone/tofu/csrf"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/size"
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
			if httpx.IsTLS(r) {
				name = CSRFTokenCookieName
			}

			var cookieToken []byte
			cookie, err := r.Cookie(name)
			if err != nil && !errors.Is(err, http.ErrNoCookie) {
				err = fmt.Errorf("read cookie: %w", err)

				handleError(w, r, err, errorHandler, http.StatusInternalServerError)

				return
			}
			if err == nil {
				decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
				if err != nil {
					err = fmt.Errorf("decode cookie token: %w", err)

					handleError(w, r, err, errorHandler, http.StatusInternalServerError)

					return
				}

				cookieToken = decoded
			}

			ctx := r.Context()
			ctx, err = csrf.SetToken(ctx, cookieToken)
			if err != nil {
				err = fmt.Errorf("CSRF set token: %w", err)

				handleError(w, r, err, errorHandler, http.StatusInternalServerError)

				return
			}

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				// Do nothing for safe methods

			default:
				var mustRenew bool
				sentToken := r.Header.Get(CSRFTokenHeaderName)
				if sentToken == "" {
					// Although not technically needed, parsing the form as a separate
					// here step allows us to handle any body reader errors, like a max bytes
					// error in normal posts
					//
					// r.ParseForm is idempotent so there's no harm in just calling it here
					if err := r.ParseForm(); err != nil {
						err = fmt.Errorf("parse form: %w", err)

						handleError(w, r, err, errorHandler, http.StatusInternalServerError)

						return
					}

					sentToken = r.PostFormValue(CSRFTokenFieldName)
					if sentToken == "" && r.MultipartForm == nil {
						contentType := r.Header.Get("content-type")
						mediaType, _, err := mime.ParseMediaType(contentType)
						if err == nil && (mediaType == "multipart/form-data" || mediaType == "multipart/mixed") {
							// If the sent token is still empty then we might be receiving a
							// multipart form, in which case we do allow for the CSRF token
							// to be read from the query string, but require that it be renewed
							// immediately in case of accidental leaks
							//
							// We prefer to do this so that the middleware doesn't force a read
							// of the multipart form body when a handler may want to stream it with a
							// multipart reader instead, especially for things like large file uploads
							sentToken = r.URL.Query().Get(CSRFTokenFieldName)
							if sentToken != "" {
								mustRenew = true
							} else {
								// If all else fails then we're forced to read the multipart
								// form body to get the CSRF token
								const maxMemory = 32 * size.Megabyte
								if err := r.ParseMultipartForm(maxMemory); err != nil {
									err = fmt.Errorf("parse multipart form: %w", err)

									handleError(w, r, err, errorHandler, http.StatusInternalServerError)

									return
								}

								sentToken = r.PostFormValue(CSRFTokenFieldName)
							}
						}
					}

				}
				if sentToken == "" {
					handleError(w, r, csrf.ErrEmptyToken, errorHandler, http.StatusBadRequest)

					return
				}

				decoded, err := base64.RawURLEncoding.DecodeString(sentToken)
				if err != nil {
					err = fmt.Errorf("decode sent token: %w", err)

					handleError(w, r, err, errorHandler, http.StatusInternalServerError)

					return
				}

				err = csrf.Check(ctx, decoded)
				if err != nil {
					err = fmt.Errorf("CSRF check: %w", err)

					handleError(w, r, err, errorHandler, http.StatusInternalServerError)

					return
				}

				if mustRenew {
					if err := csrf.RenewToken(ctx); err != nil {
						err = fmt.Errorf("CSRF renew token: %w", err)

						handleError(w, r, err, errorHandler, http.StatusInternalServerError)

						return
					}
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
				insecure:       !httpx.IsTLS(r),
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
