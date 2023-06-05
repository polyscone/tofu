package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/account"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/admin"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/page"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

const (
	publicDir   = "public"
	templateDir = "template"
)

//go:embed "public"
//go:embed "template"
var files embed.FS

func NewRouter(tenant *handler.Tenant) http.Handler {
	publicFiles := fstack.New(dev.RelDirFS(publicDir), errors.Must(fs.Sub(files, publicDir)))
	templateFiles := fstack.New(dev.RelDirFS(templateDir), errors.Must(fs.Sub(files, templateDir)))

	mux := router.NewServeMux()
	h := handler.New(mux, tenant, templateFiles)
	guard := handler.NewGuard(h, func() string {
		return mux.Path("account.sign_in")
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		h.ErrorView(w, r, errors.Tracef(err), "error", nil)
	}

	// Middleware
	mux.Use(middleware.Recover(errorHandler))
	mux.Use(middleware.Timeout(5*time.Second, errorHandler))
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
		ErrorHandler:   errorHandler,
		TrustedProxies: tenant.Proxies,
	}))
	mux.Use(middleware.Session(h.Sessions, &middleware.SessionConfig{
		Insecure:     tenant.Insecure,
		ErrorHandler: errorHandler,
	}))
	mux.Use(httputil.TraceRequest(h.Sessions, errorHandler))
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(middleware.CSRF(&middleware.CSRFConfig{
		Insecure:     tenant.Insecure,
		ErrorHandler: errorHandler,
	}))
	mux.Use(middleware.Heartbeat("/meta/health"))
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))
	mux.Use(guard.Middleware)
	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// The redirect key in the session is supposed to be a one-time temporary
			// redirect target, so we ensure it's deleted if we're visiting the target
			if h.Sessions.GetString(ctx, sess.Redirect) == r.URL.String() {
				h.Sessions.Delete(ctx, sess.Redirect)
			}

			next(w, r)
		}
	})

	// Event listeners
	tenant.Broker.Listen(accountSignedInWithPasswordHandler(tenant, h))
	tenant.Broker.Listen(accountDisabledTOTPHandler(tenant, h))
	tenant.Broker.Listen(accountSignedUpHandler(tenant, h))

	// Redirects
	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusMovedPermanently)

	// Rewrites
	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	// Pages
	page.Home(h, guard, mux)

	// Account
	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Name("account.section")

		account.Activate(h, guard, mux)
		account.ChangePassword(h, guard, mux)
		account.Dashboard(h, guard, mux)
		account.ResetPassword(h, guard, mux)
		account.SignUp(h, guard, mux)
		account.SignIn(h, guard, mux)
		account.SignOut(h, guard, mux)
		account.TOTP(h, guard, mux)
	})

	// Admin
	mux.Prefix("/admin", func(mux *router.ServeMux) {
		guard.RequireSignIn(mux.CurrentPrefix())

		mux.Name("admin.section")

		admin.Dashboard(h, guard, mux)

		mux.Prefix("/account", func(mux *router.ServeMux) {
			account.RoleManagement(h, guard, mux)
			account.UserManagement(h, guard, mux)
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
				h.ErrorView(w, r, errors.Tracef(httputil.ErrNotFound, "%w: %v %v", err, r.Method, r.URL), "error", nil)

			default:
				h.ErrorView(w, r, errors.Tracef(httputil.ErrInternalServerError, "%w: %v %v", err, r.Method, r.URL), "error", nil)
			}

			return
		}
		if stat.IsDir() {
			h.ErrorView(w, r, errors.Tracef("%w: %v %v", httputil.ErrForbidden, r.Method, r.URL), "error", nil)

			return
		}

		fileServer.ServeHTTP(w, r)
	}))

	// Generic not found handler
	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.ErrorView(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL), "error", nil)
	})

	// Generic method not allowed handler
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		h.ErrorView(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL), "error", nil)
	})

	return mux
}
