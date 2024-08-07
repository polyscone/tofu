package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/size"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/api/account"
	"github.com/polyscone/tofu/web/api/meta"
	"github.com/polyscone/tofu/web/api/security"
	"github.com/polyscone/tofu/web/api/system"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/sess"
)

func NewAPIRouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()

	mux.BasePath = app.BasePath + "/api/v1"

	h := api.NewHandler(base)

	timeoutErrorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		rc := http.NewResponseController(w)

		// Since this is the handler for a timeout we could be quite close to the
		// write deadline for the underlying TCP/IP connection, so we should extend
		// it to ensure we have enough time to write any response
		rc.SetWriteDeadline(time.Now().Add(3 * time.Second))

		if errors.Is(err, context.Canceled) {
			w.WriteHeader(httpx.StatusClientClosedRequest)

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

	mux.Use(middleware.Recover(&middleware.RecoverConfig{
		ErrorHandler: errorHandler("recover middleware"),
		Logger:       logger,
	}))
	mux.Use(middleware.Metrics(h.Metrics, "requests.API"))

	if len(h.IPWhitelist) > 0 {
		mux.Use(middleware.IPWhitelist(&middleware.IPWhitelistConfig{
			ErrorHandler:   errorHandler("ip whitelist middleware"),
			IPs:            h.IPWhitelist,
			TrustedProxies: h.Proxies,
		}))
	}

	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.NoContent)
	mux.Use(h.AttachContextLogger)
	mux.Use(middleware.SecurityHeaders(&middleware.SecurityHeadersConfig{Logger: logger}))
	mux.Use(middleware.ETag(&middleware.ETagConfig{Logger: logger}))
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
		Consume: func(r *http.Request) bool {
			whitelist := []string{".js"}

			return !slices.Contains(whitelist, filepath.Ext(r.URL.Path))
		},
		ErrorHandler:   errorHandler("rate limit middleware"),
		TrustedProxies: h.Proxies,
	}))
	mux.Use(middleware.Timeout(HandlerTimeout, &middleware.TimeoutConfig{
		ErrorHandler: timeoutErrorHandler,
		Logger:       logger,
	}))
	mux.Use(middleware.Session(h.Sessions, errorHandler("session middleware")))
	mux.Use(h.AttachContext)
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))

	// CSRF must come after max bytes middleware because it could read the request
	// body which the max bytes middleware needs to wrap first
	mux.Use(middleware.CSRF(errorHandler("CSRF middleware")))

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
		if allowedMethods, notAllowed := httpx.MethodNotAllowed(mux, r); notAllowed {
			w.Header().Set("allow", strings.Join(allowedMethods, ", "))

			h.ErrorJSON(w, r, "handler", httpx.ErrMethodNotAllowed)

			return
		}

		h.ErrorJSON(w, r, "handler", httpx.ErrNotFound)
	})

	return mux
}
