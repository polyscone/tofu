package web

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/adapter/web/ui/pwa/event"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

func NewPWARouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()
	h := ui.NewHandler(base, mux, func() string {
		return "/sign-in"
	})

	h.Broker.Listen(event.SignedInWithPasswordHandler(h))

	routePrefix := "#!"
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
		ErrorHandler: errorHandler("timeout middleware"),
		Logger:       logger,
	}))
	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.MethodOverride)
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders(&middleware.SecurityHeadersConfig{Logger: logger}))
	mux.Use(middleware.ETag(&middleware.ETagConfig{Logger: logger}))
	mux.Use(middleware.Session(h.Sessions, &middleware.SessionConfig{
		Insecure:     h.Insecure,
		ErrorHandler: errorHandler("session middleware"),
	}))
	mux.Use(h.AttachContext)
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
	mux.Use(middleware.CSRF(&middleware.CSRFConfig{
		Insecure:     h.Insecure,
		ErrorHandler: errorHandler("CSRF middleware"),
	}))
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
		Consume: func(r *http.Request) bool {
			whitelist := []string{".css", ".gif", ".ico", ".jpeg", ".jpg", ".js", ".png"}

			return !slices.Contains(whitelist, filepath.Ext(r.URL.Path))
		},
		ErrorHandler:   errorHandler("rate limit middleware"),
		TrustedProxies: h.Proxies,
	}))
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))

	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusMovedPermanently)

	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	mux.Get("/robots.txt", h.Plain.Handler("file/robots"))
	mux.Get("/.well-known/security.txt", h.Plain.Handler("file/security"))
	mux.Get("/app.webmanifest", h.JSON.Handler("file/pwa_webmanifest"))

	rootVars := func(h *ui.Handler, r *http.Request) handler.Vars {
		ctx := r.Context()
		config := h.Config(ctx)

		return handler.Vars{
			"url": r.URL.String(),
			"config": map[string]any{
				"prefix":               routePrefix,
				"signUpEnabled":        config.SignUpEnabled,
				"googleSignInEnabled":  config.GoogleSignInEnabled,
				"googleSignInClientId": config.GoogleSignInClientID,
			},
			"vars": map[string]any{
				"siteHostname": h.Tenant.Vars["siteHostname"],
			},
		}
	}

	publicFilesRoot := http.FS(publicFiles)
	fileServer := http.FileServer(publicFilesRoot)
	mux.GetHandler("/{rest...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)

		logger.Error("handler", "error", httputil.ErrNotFound)

		http.Redirect(w, r, routePrefix+"/error/404", http.StatusSeeOther)
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)

		logger.Error("handler", "error", httputil.ErrMethodNotAllowed)

		http.Redirect(w, r, routePrefix+"/error/405", http.StatusSeeOther)
	})

	return mux
}
