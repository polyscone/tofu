package site

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/internal/size"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/event"
	"github.com/polyscone/tofu/web/site/route/account"
	"github.com/polyscone/tofu/web/site/route/admin"
	"github.com/polyscone/tofu/web/site/route/page"
	"github.com/polyscone/tofu/web/site/route/system"
	"github.com/polyscone/tofu/web/site/ui"
)

func NewRouter(base *handler.Handler, handlerTimeout time.Duration) http.Handler {
	mux := router.NewServeMux()

	mux.BasePath = app.BasePath

	h := ui.NewHandler(base, mux, func() string {
		return mux.Path("account.sign_in")
	})

	h.Broker.Listen(event.ActivatedHandler(h))
	h.Broker.Listen(event.AlreadySignedUpHandler(h))
	h.Broker.Listen(event.InvitedHandler(h))
	h.Broker.Listen(event.PasswordResetRequestedHandler(h))
	h.Broker.Listen(event.SignInMagicLinkRequestedHandler(h))
	h.Broker.Listen(event.SignedInHandler(h))
	h.Broker.Listen(event.SignedUpHandler(h))
	h.Broker.Listen(event.TOTPDisabledHandler(h))
	h.Broker.Listen(event.TOTPSMSRequestedHandler(h))

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

		h.HTML.ErrorView(w, r, "timeout middleware", err, "error", nil)
	}
	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			h.HTML.ErrorView(w, r, msg, err, "error", nil)
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
	mux.Use(middleware.Metrics(h.Metrics, "requests.Site"))

	if len(h.IPWhitelist) > 0 {
		mux.Use(middleware.IPWhitelist(&middleware.IPWhitelistConfig{
			ErrorHandler:   errorHandler("ip whitelist middleware"),
			IPs:            h.IPWhitelist,
			TrustedProxies: h.Proxies,
		}))
	}

	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.MethodOverride)
	mux.Use(middleware.NoContent)
	mux.Use(h.AttachContextLogger)
	mux.Use(middleware.SecurityHeaders(&middleware.SecurityHeadersConfig{Logger: logger}))
	mux.Use(middleware.ETag(&middleware.ETagConfig{Logger: logger}))
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
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
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		csrf := middleware.CSRF(errorHandler("CSRF middleware"))(next)

		return func(w http.ResponseWriter, r *http.Request) {
			// Google sign in provides its own CSRF token which is checked in the POST handler
			if r.URL.Path == mux.Path("account.sign_in.google.post") {
				next(w, r)
			} else {
				csrf(w, r)
			}
		}
	})

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

				h.AddFlashErrorf(ctx, i18n.M("site:flash.unauthorised"))

				http.Redirect(w, r, mux.Path("page.home"), http.StatusSeeOther)

				return
			}

			next(w, r)
		}
	})
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		var setupDone atomic.Bool

		return func(w http.ResponseWriter, r *http.Request) {
			if filepath.Ext(r.URL.Path) != "" {
				next(w, r)

				return
			}

			ctx := r.Context()
			config := h.Config(ctx)
			user := h.User(ctx)

			if !setupDone.Load() {
				userCount, err := h.Repo.Account.CountUsers(ctx)
				if err != nil {
					h.HTML.ErrorView(w, r, "count users", err, "error", nil)

					return
				}

				setupDone.Store(!config.SetupRequired && userCount != 0)

				systemSetupPath := mux.Path("system.setup")
				if r.Method == http.MethodGet && !setupDone.Load() && r.URL.Path != systemSetupPath {
					http.Redirect(w, r, systemSetupPath, http.StatusSeeOther)

					return
				}
			}

			isTOTPSection := h.HasPathPrefix(r.URL.Path, "account.totp.section")
			isChoosePasswordSection := h.HasPathPrefix(r.URL.Path, "account.choose_password.section")
			isSignOut := r.URL.Path == h.Path("account.sign_out.post")
			isAllowedPath := isTOTPSection || isChoosePasswordSection || isSignOut
			isSignedIn := h.Session.IsSignedIn(ctx)
			if isSignedIn && config.TOTPRequired && !user.HasActivatedTOTP() && !isAllowedPath {
				h.AddFlashf(ctx, i18n.M("flash.totp_required"))

				http.Redirect(w, r, mux.Path("account.totp.setup"), http.StatusSeeOther)

				return
			}

			next(w, r)
		}
	})

	page.RegisterHandlers(h, mux)

	mux.Named("account.section", "/account")

	account.RegisterChangePasswordHandlers(h, mux)
	account.RegisterChoosePasswordHandlers(h, mux)
	account.RegisterDashboardHandlers(h, mux)
	account.RegisterResetPasswordHandlers(h, mux)
	account.RegisterRoleManagementHandlers(h, mux)
	account.RegisterSignInHandlers(h, mux)
	account.RegisterSignOutHandlers(h, mux)
	account.RegisterSignUpHandlers(h, mux)
	account.RegisterTOTPHandlers(h, mux)
	account.RegisterUserManagementHandlers(h, mux)
	account.RegisterVerifyHandlers(h, mux)

	mux.Named("admin.section", "/admin")

	admin.RegisterDashboardHandlers(h, mux)

	system.RegisterConfigHandlers(h, mux)
	system.RegisterMetricsHandlers(h, mux)
	system.RegisterSetupHandlers(h, mux)

	mux.Handle("/security.txt", http.RedirectHandler("/.well-known/security.txt", http.StatusMovedPermanently))

	mux.Handle("/favicon.ico", httpx.RewriteHandler(mux, "/favicon.png"))

	renderer := handler.NewRenderer(handler.RendererConfig{
		Handler:         h.Handler,
		AssetTags:       ui.AssetTags,
		AssetFiles:      ui.AssetFiles,
		Funcs:           h.Funcs,
		WrapI18nRuntime: ui.NewI18nRuntimeWrapper(h),
	})
	serveFile := handler.NewFileServer(mux, renderer, func(w http.ResponseWriter, r *http.Request, err error) {
		h.HTML.ErrorView(w, r, "static file", err, "error", nil)
	})
	mux.HandleFunc("/", serveFile)

	return mux
}
