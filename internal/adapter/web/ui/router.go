package ui

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/account"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/admin"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/page"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
	"golang.org/x/exp/slices"
)

const (
	publicDir   = "public"
	templateDir = "template"
)

//go:embed "public"
//go:embed "template"
var files embed.FS

func NewRouter(tenant *handler.Tenant) http.Handler {
	publicFiles := fstack.New(dev.RelDirFS(publicDir), errsx.Must(fs.Sub(files, publicDir)))
	templateFiles := fstack.New(dev.RelDirFS(templateDir), errsx.Must(fs.Sub(files, templateDir)))

	mux := router.NewServeMux()
	h := handler.New(mux, tenant, templateFiles, "account.sign_in", "system.config")

	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			h.ErrorView(w, r, msg, err, "error", nil)
		}
	}

	// Middleware
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
		TrustedProxies: tenant.Proxies,
	}))
	mux.Use(middleware.Session(h.Sessions, &middleware.SessionConfig{
		Insecure:     tenant.Insecure,
		ErrorHandler: errorHandler("session middleware"),
	}))
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		csrf := middleware.CSRF(&middleware.CSRFConfig{
			Insecure:     tenant.Insecure,
			ErrorHandler: errorHandler("CSRF middleware"),
		})

		return func(w http.ResponseWriter, r *http.Request) {
			exceptions := []string{
				mux.Path("account.sign_in.google.post"),
			}

			if slices.Contains(exceptions, r.URL.Path) {
				next(w, r)
			} else {
				csrf(next)(w, r)
			}
		}
	})
	mux.Use(middleware.Heartbeat("/meta/health"))
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))
	mux.Use(h.Middleware)

	// Event listeners
	tenant.Broker.Listen(accountSignedInWithPasswordHandler(h))
	tenant.Broker.Listen(accountDisabledTOTPHandler(h))
	tenant.Broker.Listen(accountSignedUpHandler(h))

	// Redirects
	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusMovedPermanently)

	// Rewrites
	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	// Pages
	page.Home(h, mux)

	// Account
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

	// Admin
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

	// Public static file handler
	publicFilesRoot := http.FS(publicFiles)
	fileServer := http.FileServer(publicFilesRoot)
	mux.GetHandler("/:rest*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		upath = path.Clean(upath)

		stat, err := fs.Stat(files, publicDir+upath)
		if err != nil {
			switch {
			case errors.Is(err, fs.ErrNotExist):
				h.ErrorView(w, r, "static file", fmt.Errorf("%w: %w", httputil.ErrNotFound, err), "error", nil)

			default:
				h.ErrorView(w, r, "static file", fmt.Errorf("%w: %w", httputil.ErrInternalServerError, err), "error", nil)
			}

			return
		}
		if stat.IsDir() {
			h.ErrorView(w, r, "static directory", httputil.ErrForbidden, "error", nil)

			return
		}

		fileServer.ServeHTTP(w, r)
	}))

	// Generic not found handler
	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.ErrorView(w, r, "handler", httputil.ErrNotFound, "error", nil)
	})

	// Generic method not allowed handler
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		h.ErrorView(w, r, "handler", httputil.ErrMethodNotAllowed, "error", nil)
	})

	return mux
}
