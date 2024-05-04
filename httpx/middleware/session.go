package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/session"
)

const (
	SessionCookieName         = "__Host-session"
	SessionCookieNameInsecure = "session"
)

func Session(sm *session.Manager, errorHandler ErrorHandler) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookieSessionID, err := getSessionCookieID(r)
			if err != nil {
				err = fmt.Errorf("get session cookie id: %w", err)

				handleError(w, r, err, errorHandler, http.StatusInternalServerError)

				return
			}

			ctx, err := sm.Load(r.Context(), cookieSessionID)
			if err != nil {
				err = fmt.Errorf("session manager: load: %w", err)

				handleError(w, r, err, errorHandler, http.StatusInternalServerError)

				return
			}

			found := slices.ContainsFunc(w.Header().Values("vary"), func(el string) bool {
				return strings.ToLower(el) == "cookie"
			})
			if !found {
				w.Header().Add("vary", "cookie")
			}

			rw := &sessionResponseWriter{
				ResponseWriter:  w,
				request:         r,
				errorHandler:    errorHandler,
				insecure:        !httpx.IsTLS(r),
				sm:              sm,
				ctx:             ctx,
				cookieSessionID: cookieSessionID,
			}
			r = r.WithContext(ctx)

			next(rw, r)

			rw.commit()
		}
	}
}

var _ Unwrapper = (*sessionResponseWriter)(nil)

type sessionResponseWriter struct {
	http.ResponseWriter
	request         *http.Request
	errorHandler    ErrorHandler
	insecure        bool
	sm              *session.Manager
	ctx             context.Context
	cookieSessionID string
	committed       bool
}

func (w *sessionResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *sessionResponseWriter) Write(b []byte) (int, error) {
	w.commit()

	return w.ResponseWriter.Write(b)
}

func (w *sessionResponseWriter) WriteHeader(statusCode int) {
	w.commit()

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sessionResponseWriter) commit() {
	if w.committed {
		return
	}

	w.committed = true

	id, err := w.sm.Commit(w.ctx)
	if handleError(w, w.request, err, w.errorHandler, http.StatusInternalServerError) {
		return
	}

	name := SessionCookieName
	if w.insecure {
		name = SessionCookieNameInsecure
	}

	switch {
	case w.sm.Status(w.ctx) == session.Destroyed:
		http.SetCookie(w.ResponseWriter, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1000,
			Expires:  time.Now().Add(-1 * time.Hour),
			HttpOnly: true,
			Secure:   !w.insecure,
			SameSite: http.SameSiteLaxMode,
		})

	case w.cookieSessionID != id:
		http.SetCookie(w.ResponseWriter, &http.Cookie{
			Name:     name,
			Value:    id,
			Path:     "/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   !w.insecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func getSessionCookieID(r *http.Request) (string, error) {
	name := SessionCookieNameInsecure
	if httpx.IsTLS(r) {
		name = SessionCookieName
	}

	cookie, err := r.Cookie(name)
	if errors.Is(err, http.ErrNoCookie) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("session cookie id: %w", err)
	}

	return cookie.Value, nil
}
