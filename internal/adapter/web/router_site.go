package web

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/adapter/web/ui/account"
	"github.com/polyscone/tofu/internal/adapter/web/ui/admin"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

func NewSiteRouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()
	h := ui.NewHandler(base, mux, func() string {
		return mux.Path("account.sign_in")
	})

	h.Broker.Listen(account.SignedInWithPasswordHandler(h))
	h.Broker.Listen(account.SignedUpHandler(h))
	h.Broker.Listen(account.TOTPDisabledHandler(h))

	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			h.HTML.ErrorView(w, r, msg, err, "site/error", nil)
		}
	}

	mux.Use(middleware.Recover(errorHandler("recover middleware")))
	mux.Use(middleware.Timeout(5*time.Second, errorHandler("timeout middleware")))
	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.MethodOverride)
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
		Consume: func(r *http.Request) bool {
			whitelist := []string{
				".css",
				".gif",
				".ico",
				".jpeg",
				".jpg",
				".js",
				".png",
			}

			for _, ext := range whitelist {
				if strings.HasSuffix(r.URL.Path, ext) {
					return false
				}
			}

			return true
		},
		ErrorHandler:   errorHandler("rate limit middleware"),
		TrustedProxies: h.Proxies,
	}))
	mux.Use(middleware.Session(h.Sessions, &middleware.SessionConfig{
		Insecure:     h.Insecure,
		ErrorHandler: errorHandler("session middleware"),
	}))
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		csrf := middleware.CSRF(&middleware.CSRFConfig{
			Insecure:     h.Insecure,
			ErrorHandler: errorHandler("CSRF middleware"),
		})

		return func(w http.ResponseWriter, r *http.Request) {
			// Google sign in provides its own CSRF token which is checked
			// in the POST handler
			if r.URL.Path == mux.Path("account.sign_in.google.post") {
				next(w, r)
			} else {
				csrf(next)(w, r)
			}
		}
	})
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))
	mux.Use(h.AttachContext)
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			config := h.Config(ctx)
			user := h.User(ctx)

			isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

			systemConfigPath := mux.Path("system.config")
			if r.Method == http.MethodGet && config.RequireSetup && r.URL.Path != systemConfigPath && filepath.Ext(r.URL.Path) == "" {
				http.Redirect(w, r, systemConfigPath, http.StatusSeeOther)

				return
			}

			isInTOTPSection := h.HasPathPrefix(r.URL.Path, "account.totp.section")
			if !isInTOTPSection && isSignedIn && config.RequireTOTP && !user.HasActivatedTOTP() {
				h.AddFlashf(ctx, "Two-factor authentication is required to use this application.")

				http.Redirect(w, r, mux.Path("account.totp.setup"), http.StatusSeeOther)

				return
			}

			next(w, r)
		}
	})

	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusMovedPermanently)

	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	mux.Get("/robots.txt", h.Plain.Handler("file/robots"))
	mux.Get("/.well-known/security.txt", h.Plain.Handler("file/security"))

	mux.Get("/", h.HTML.Handler("site/page/home"), "page.home")

	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Name("account.section")

		account.Activate(h, mux)
		account.ChangePassword(h, mux)
		account.ChoosePassword(h, mux)
		account.Dashboard(h, mux)
		account.ResetPassword(h, mux)
		account.SignUp(h, mux)
		account.SignIn(h, mux)
		account.SignOut(h, mux)
		account.TOTP(h, mux)
	})

	mux.Prefix("/admin", func(mux *router.ServeMux) {
		mux.Name("admin.section")

		admin.Dashboard(h, mux)

		mux.Prefix("/account", func(mux *router.ServeMux) {
			mux.Before(h.RequireSignIn)

			account.RoleManagement(h, mux)
			account.UserManagement(h, mux)
		})

		mux.Prefix("/system", func(mux *router.ServeMux) {
			mux.Before(h.RequireSignInIf(func(p guard.Passport) bool { return !p.System.CanViewConfig() }))
			mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.System.CanViewConfig() }))

			admin.SystemConfig(h, mux)
		})
	})

	publicFilesRoot := http.FS(publicFiles)
	fileServer := http.FileServer(publicFilesRoot)
	mux.GetHandler("/:rest*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		upath = path.Clean(upath)

		stat, err := fs.Stat(publicFiles, strings.TrimPrefix(upath, "/"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
				h.HTML.ErrorView(w, r, "static file", fmt.Errorf("%w: %w", httputil.ErrNotFound, err), "site/error", nil)
			} else {
				h.HTML.ErrorView(w, r, "static file", fmt.Errorf("%w: %w", httputil.ErrInternalServerError, err), "site/error", nil)
			}

			return
		}
		if stat.IsDir() {
			h.HTML.ErrorView(w, r, "static file", httputil.ErrForbidden, "site/error", nil)

			return
		}

		fileServer.ServeHTTP(w, r)
	}))

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.HTML.ErrorView(w, r, "handler", httputil.ErrNotFound, "site/error", nil)
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		h.HTML.ErrorView(w, r, "handler", httputil.ErrMethodNotAllowed, "site/error", nil)
	})

	return mux
}
