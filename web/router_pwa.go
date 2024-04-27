package web

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/polyscone/tofu/pkg/http/middleware"
	"github.com/polyscone/tofu/pkg/http/router"
	"github.com/polyscone/tofu/pkg/size"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/httputil"
	"github.com/polyscone/tofu/web/sess"
	"github.com/polyscone/tofu/web/ui"
	"github.com/polyscone/tofu/web/ui/pwa/event"
)

func NewPWARouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()
	h := ui.NewHandler(base, mux, func() string {
		return "/sign-in"
	})

	h.Broker.Listen(event.AlreadySignedUpHandler(h))
	h.Broker.Listen(event.PasswordResetRequestedHandler(h))
	h.Broker.Listen(event.SignInMagicLinkRequestedHandler(h))
	h.Broker.Listen(event.SignedInHandler(h))
	h.Broker.Listen(event.SignedUpHandler(h))
	h.Broker.Listen(event.TOTPSMSRequestedHandler(h))

	routePrefix := "#!"
	timeoutErrorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if errors.Is(err, context.Canceled) {
			w.WriteHeader(httputil.StatusClientClosedRequest)

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

	mux.Use(middleware.Recover(errorHandler("recover middleware")))
	mux.Use(middleware.Metrics(h.Metrics, "requests.PWA"))
	mux.Use(h.AttachContextLogger)
	mux.Use(middleware.Timeout(HandlerTimeout, &middleware.TimeoutConfig{
		ErrorHandler: timeoutErrorHandler,
		Logger:       logger,
	}))
	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.MethodOverride)
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
		Consume: func(r *http.Request) bool {
			whitelist := []string{".css", ".gif", ".ico", ".jpeg", ".jpg", ".js", ".png"}

			return !slices.Contains(whitelist, filepath.Ext(r.URL.Path))
		},
		ErrorHandler:   errorHandler("rate limit middleware"),
		TrustedProxies: h.Proxies,
	}))

	mux.HandleFunc("GET /robots.txt", h.Plain.HandlerFunc("file/robots"))
	mux.HandleFunc("GET /.well-known/security.txt", h.Plain.HandlerFunc("file/security"))
	mux.HandleFunc("GET /pwa.webmanifest", h.JSON.HandlerFunc("file/pwa_webmanifest"))

	mux.Handle("/security.txt", http.RedirectHandler("/.well-known/security.txt", http.StatusMovedPermanently))

	mux.Handle("/favicon.ico", httputil.RewriteHandler(mux, "/favicon.png"))

	rootVars := func(h *ui.Handler, r *http.Request) handler.Vars {
		ctx := r.Context()
		config := h.Config(ctx)

		return handler.Vars{
			"url": r.URL.String(),
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

	publicFilesRoot := http.FS(publicFiles)
	fileServer := http.FileServer(publicFilesRoot)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if allowed, ok := httputil.MethodNotAllowed(mux, r); ok {
			w.Header().Set("allow", strings.Join(allowed, ", "))

			http.Redirect(w, r, routePrefix+"/error/405", http.StatusSeeOther)

			return
		}

		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		upath = path.Clean(upath)

		stat, err := fs.Stat(publicFiles, strings.TrimPrefix(upath, "/"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
				h.HTML.View(w, r, http.StatusOK, "pwa/root", rootVars(h, r))
			} else {
				ctx := r.Context()
				logger := h.Logger(ctx)

				logger.Error("static file", "error", err)

				http.Redirect(w, r, routePrefix+"/error/500", http.StatusSeeOther)
			}

			return
		}
		if stat.IsDir() {
			h.HTML.View(w, r, http.StatusOK, "pwa/root", rootVars(h, r))

			return
		}

		fileServer.ServeHTTP(w, r)
	})

	return mux
}
