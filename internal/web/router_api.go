package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
	"github.com/polyscone/tofu/internal/web/api"
	"github.com/polyscone/tofu/internal/web/api/account"
	"github.com/polyscone/tofu/internal/web/api/meta"
	"github.com/polyscone/tofu/internal/web/api/security"
	"github.com/polyscone/tofu/internal/web/api/system"
	"github.com/polyscone/tofu/internal/web/handler"
	"github.com/polyscone/tofu/internal/web/httputil"
	"github.com/polyscone/tofu/internal/web/sess"
)

func NewAPIRouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()
	h := api.NewHandler(base)

	timeoutErrorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if errors.Is(err, context.Canceled) {
			w.WriteHeader(httputil.StatusClientClosedRequest)

			return
		}

		h.ErrorJSON(w, r, "timeout middleware", err)
	}
	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			h.ErrorJSON(w, r, msg, err)
		}
	}
	logger := func(r *http.Request) *slog.Logger {
		ctx := r.Context()

		return h.Logger(ctx)
	}

	mux.Use(middleware.Recover(errorHandler("recover middleware")))
	mux.Use(middleware.Metrics(h.Metrics, "requests.API"))
	mux.Use(h.AttachContextLogger)
	mux.Use(middleware.Timeout(HandlerTimeout, &middleware.TimeoutConfig{
		ErrorHandler: timeoutErrorHandler,
		Logger:       logger,
	}))
	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders(&middleware.SecurityHeadersConfig{Logger: logger}))
	mux.Use(middleware.ETag(&middleware.ETagConfig{Logger: logger}))
	mux.Use(middleware.Session(h.Sessions, errorHandler("session middleware")))
	mux.Use(h.AttachContext)
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user := h.User(ctx)

			isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)
			if isSignedIn && user.IsSuspended() {
				logger := h.Logger(ctx)

				logger.Info("user forcibly signed out due to suspension of account")

				h.Sessions.Clear(ctx)
				h.Sessions.Renew(ctx)
			}

			next(w, r)
		}
	})
	mux.Use(middleware.CSRF(errorHandler("CSRF middleware")))
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
		ErrorHandler:   errorHandler("rate limit middleware"),
		TrustedProxies: h.Proxies,
	}))

	mux.HandleFunc("GET /sdk.js", h.JavaScript.HandlerFunc("sdk/v1.js"))

	account.RegisterResetPasswordHandlers(h, mux)
	account.RegisterSessionHandlers(h, mux)
	account.RegisterSignInHandlers(h, mux)
	account.RegisterSignOutHandlers(h, mux)
	account.RegisterSignUpHandlers(h, mux)
	account.RegisterVerifyHandlers(h, mux)

	meta.RegisterHealthHandlers(h, mux)

	security.RegisterCSRFHandlers(h, mux)

	system.RegisterConfigHandlers(h, mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if allowed, ok := httputil.MethodNotAllowed(mux, r); ok {
			w.Header().Set("allow", strings.Join(allowed, ", "))

			h.ErrorJSON(w, r, "handler", httputil.ErrMethodNotAllowed)

			return
		}

		h.ErrorJSON(w, r, "handler", httputil.ErrNotFound)
	})

	return mux
}
