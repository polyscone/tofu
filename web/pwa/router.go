package pwa

import (
	"context"
	"errors"
	"io/fs"
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
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/pwa/event"
	"github.com/polyscone/tofu/web/pwa/ui"
)

func NewRouter(base *handler.Handler, handlerTimeout time.Duration, config handler.RouterConfig) http.Handler {
	mux := router.NewServeMux()

	mux.BasePath = app.BasePath

	h := ui.NewHandler(base, mux, func() string {
		return "/sign-in"
	})

	h.Broker.Listen(event.AccountAlreadySignedUpHandler(h))
	h.Broker.Listen(event.AccountSignedInHandler(h))
	h.Broker.Listen(event.AccountSignedUpHandler(h))

	h.Broker.Listen(event.WebPasswordResetRequestedHandler(h))
	h.Broker.Listen(event.WebSignInMagicLinkRequestedHandler(h))
	h.Broker.Listen(event.WebTOTPSMSRequestedHandler(h))
	h.Broker.ListenAny(event.WebAnyHandler(h))

	routePrefix := mux.BasePath
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

		ctx := r.Context()
		logger := h.Logger(ctx)

		logger.Error("timeout middleware", "error", err)

		http.Redirect(w, r, routePrefix+"/error/500", http.StatusSeeOther)
	}
	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			ctx := r.Context()
			logger := h.Logger(ctx)

			logger.Error(msg, "error", err)

			http.Redirect(w, r, routePrefix+"/error/500", http.StatusSeeOther)
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
	mux.Use(middleware.Metrics(h.Metrics, "requests.PWA"))

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
	mux.Use(middleware.RateLimit(config.RateLimit.Capacity, config.RateLimit.Replenish, &middleware.RateLimitConfig{
		Consume: func(r *http.Request) bool {
			whitelist := []string{".css", ".gif", ".ico", ".jpeg", ".jpg", ".js", ".png", ".webp"}

			return !slices.Contains(whitelist, filepath.Ext(r.URL.Path))
		},
		ErrorHandler:   errorHandler("rate limit middleware"),
		TrustedProxies: h.Proxies,
	}))
	mux.Use(middleware.Timeout(handlerTimeout, &middleware.TimeoutConfig{
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

	mux.Handle("/security.txt", http.RedirectHandler("/.well-known/security.txt", http.StatusMovedPermanently))

	mux.Handle("/favicon.ico", httpx.RewriteHandler(mux, "/favicon.png"))

	rootVars := func(h *ui.Handler, r *http.Request) handler.Vars {
		ctx := r.Context()
		config := h.Config(ctx)

		url := r.URL.String()
		if mux.BasePath != "" {
			url = strings.TrimPrefix(url, mux.BasePath)
		}
		if url == "" {
			url = "/"
		}

		return handler.Vars{
			"url": url,
			"config": map[string]any{
				"prefix":                 routePrefix,
				"signUpEnabled":          config.SignUpEnabled,
				"magicLinkSignInEnabled": config.MagicLinkSignInEnabled,
				"googleSignInEnabled":    config.GoogleSignInEnabled,
				"googleSignInClientId":   config.GoogleSignInClientID,
				"facebookSignInEnabled":  config.FacebookSignInEnabled,
				"facebookSignInAppId":    config.FacebookSignInAppID,
			},
		}
	}

	renderer := handler.NewRenderer(handler.RendererConfig{
		Handler:         h.Handler,
		AssetTags:       ui.AssetTags,
		AssetFiles:      ui.AssetFiles,
		Funcs:           h.Funcs,
		T:               h.T,
		WrapI18nRuntime: handler.NewI18nRuntimeWrapper(mux),
	})
	serveFile := handler.NewFileServer(mux, renderer, func(w http.ResponseWriter, r *http.Request, err error) {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) || errors.Is(err, handler.ErrNoIndex) {
			h.HTML.View(w, r, http.StatusOK, "root", rootVars(h, r))

			return
		}

		h.HTML.ErrorView(w, r, "static file", err, "error", nil)
	})
	mux.HandleFunc("/", serveFile)

	return mux
}
