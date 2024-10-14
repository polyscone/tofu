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
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/size"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/api/v1/account"
	"github.com/polyscone/tofu/web/api/v1/meta"
	"github.com/polyscone/tofu/web/api/v1/security"
	"github.com/polyscone/tofu/web/api/v1/system"
	"github.com/polyscone/tofu/web/handler"
)

func NewAPIRouterV1(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()

	mux.BasePath = app.BasePath

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
	mux.Use(middleware.Session(h.Session.Manager, errorHandler("session middleware")))
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

			isSignedIn := h.Session.IsSignedIn(ctx)
			if isSignedIn && user.IsSuspended() {
				logger := h.Logger(ctx)

				logger.Info("user forcibly signed out due to suspension of account")

				h.Session.Clear(ctx)
				h.Session.Renew(ctx)
			}

			next(w, r)
		}
	})

	account.RegisterResetPasswordHandlers(h, mux)
	account.RegisterSessionHandlers(h, mux)
	account.RegisterSignInHandlers(h, mux)
	account.RegisterSignOutHandlers(h, mux)
	account.RegisterSignUpHandlers(h, mux)
	account.RegisterVerifyHandlers(h, mux)

	meta.RegisterHealthHandlers(h, mux)

	security.RegisterCSRFHandlers(h, mux)

	system.RegisterConfigHandlers(h, mux)

	renderer := handler.NewRenderer(handler.RendererConfig{
		Handler:    h.Handler,
		AssetTags:  api.AssetTagsV1,
		AssetFiles: api.PublicFilesV1,
		Funcs:      handler.NewTemplateFuncs(nil),
	})
	serveFile := newFileServer(mux, renderer, func(w http.ResponseWriter, r *http.Request, err error) {
		h.ErrorJSON(w, r, "static file", err)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, mux.BasePath+"/api/v1")

		serveFile(w, r)
	})

	return mux
}
