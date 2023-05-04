package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/session"
)

const SessionCookieName = "__Host-session"

type SessionConfig struct {
	Insecure     bool
	ErrorHandler ErrorHandler
}

var _ http.Pusher = (*sessionResponseWriter)(nil)

type sessionResponseWriter struct {
	http.ResponseWriter
	request         *http.Request
	config          *SessionConfig
	sm              *session.Manager
	ctx             context.Context
	cookieSessionID string
	committed       bool
}

func (w *sessionResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
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
	if handleError(w, w.request, errors.Tracef(err), w.config.ErrorHandler, http.StatusInternalServerError) {
		return
	}

	switch {
	case w.sm.Status(w.ctx) == session.Destroyed:
		http.SetCookie(w.ResponseWriter, &http.Cookie{
			Name:     SessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1000,
			Expires:  time.Now().Add(-1 * time.Hour),
			HttpOnly: true,
			Secure:   !w.config.Insecure,
			SameSite: http.SameSiteLaxMode,
		})

	case w.cookieSessionID != id:
		http.SetCookie(w.ResponseWriter, &http.Cookie{
			Name:     SessionCookieName,
			Value:    id,
			Path:     "/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   !w.config.Insecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func getSessionCookieID(r *http.Request) (string, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return "", nil
	}
	if err != nil {
		return "", errors.Tracef(err)
	}

	return cookie.Value, nil
}

func Session(sm *session.Manager, config *SessionConfig) Middleware {
	if config == nil {
		config = &SessionConfig{}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookieSessionID, err := getSessionCookieID(r)
			if handleError(w, r, errors.Tracef(err), config.ErrorHandler, http.StatusInternalServerError) {
				return
			}

			ctx, err := sm.Load(r.Context(), cookieSessionID)
			if handleError(w, r, errors.Tracef(err), config.ErrorHandler, http.StatusInternalServerError) {
				return
			}

			var found bool
			for _, value := range w.Header().Values("vary") {
				if found = strings.ToLower(value) == "cookie"; found {
					break
				}
			}
			if !found {
				w.Header().Add("vary", "cookie")
			}

			rw := &sessionResponseWriter{
				ResponseWriter:  w,
				request:         r,
				config:          config,
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
