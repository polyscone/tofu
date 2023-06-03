package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
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

func NewHandler(tenant *handler.Tenant) http.Handler {
	publicFiles := fstack.New(dev.RelDirFS(publicDir), errors.Must(fs.Sub(files, publicDir)))
	templateFiles := fstack.New(dev.RelDirFS(templateDir), errors.Must(fs.Sub(files, templateDir)))

	mux := router.NewServeMux()
	svc := handler.NewServices(mux, tenant, templateFiles)
	guard := handler.NewGuard(svc, func() string {
		return mux.Path("account.sign_in")
	})

	tenant.Broker.Listen(accountSignedInWithPasswordHandler(tenant, svc))
	tenant.Broker.Listen(accountDisabledTOTPHandler(tenant, svc))
	tenant.Broker.Listen(accountSignedUpHandler(tenant, svc))

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		svc.ErrorView(w, r, errors.Tracef(err), "error", nil)
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
	mux.Use(middleware.Session(svc.Sessions, &middleware.SessionConfig{
		Insecure:     tenant.Insecure,
		ErrorHandler: errorHandler,
	}))
	mux.Use(httputil.TraceRequest(svc.Sessions, errorHandler))
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
			if svc.Sessions.GetString(ctx, sess.Redirect) == r.URL.String() {
				svc.Sessions.Delete(ctx, sess.Redirect)
			}

			next(w, r)
		}
	})

	// Redirects
	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusMovedPermanently)

	// Rewrites
	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	// Pages
	page.Home(svc, mux)

	// Account
	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Name("account.section")

		account.Activate(svc, mux)
		account.ChangePassword(svc, mux, guard)
		account.Dashboard(svc, mux, guard)
		account.ResetPassword(svc, mux)
		account.SignUp(svc, mux)
		account.SignIn(svc, mux)
		account.SignOut(svc, mux)
		account.TOTP(svc, mux, guard)
	})

	// Admin
	mux.Prefix("/admin", func(mux *router.ServeMux) {
		guard.RequireSignInPrefix(mux.CurrentPath())

		mux.Name("admin.section")

		admin.Dashboard(svc, mux)

		mux.Prefix("/account", func(mux *router.ServeMux) {
			account.RoleManagement(svc, mux, guard)
			account.UserManagement(svc, mux)
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
				svc.ErrorView(w, r, errors.Tracef(httputil.ErrNotFound, "%w: %v %v", err, r.Method, r.URL), "error", nil)

			default:
				svc.ErrorView(w, r, errors.Tracef(httputil.ErrInternalServerError, "%w: %v %v", err, r.Method, r.URL), "error", nil)
			}

			return
		}
		if stat.IsDir() {
			svc.ErrorView(w, r, errors.Tracef("%w: %v %v", httputil.ErrForbidden, r.Method, r.URL), "error", nil)

			return
		}

		fileServer.ServeHTTP(w, r)
	}))

	// Generic not found handler
	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		svc.ErrorView(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL), "error", nil)
	})

	// Generic method not allowed handler
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		svc.ErrorView(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL), "error", nil)
	})

	return mux
}
