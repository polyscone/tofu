package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/account"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/page"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

//go:embed "public"
//go:embed "template"
var files embed.FS

func NewHandler(tenant *handler.Tenant) http.Handler {
	publicFiles := fstack.New(dev.RelDirFS("public"), errors.Must(fs.Sub(files, "public")))
	templateFiles := fstack.New(dev.RelDirFS("template"), errors.Must(fs.Sub(files, "template")))

	mux := router.NewServeMux()
	svc := handler.NewServices(mux, tenant, templateFiles)
	guard := handler.NewGuard(svc, func() string {
		return svc.Path("account.login")
	})

	tenant.Broker.Listen(accountRegisteredHandler(tenant, svc))
	tenant.Broker.Listen(accountResetPasswordRequestedHandler(tenant, svc))

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		svc.ErrorView(w, r, errors.Tracef(err), "error", nil)
	}

	// Middleware
	mux.Use(middleware.Recover(errorHandler))
	mux.Use(middleware.Timeout(5*time.Second, errorHandler))
	mux.Use(middleware.MethodOverride)
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
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

	// Redirects
	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusSeeOther)

	// Rewrites
	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	// Pages
	page.Home(svc, mux)

	// Account
	mux.Prefix("/account", func(mux *router.ServeMux) {
		account.Activate(svc, mux, tenant.Tokens)
		account.ChangePassword(svc, mux, guard)
		account.Dashboard(svc, mux, guard)
		account.ResetPassword(svc, mux, tenant.Tokens)
		account.Login(svc, mux)
		account.Logout(svc, mux)
		account.Register(svc, mux)
		account.TOTP(svc, mux, guard)
	})

	// Public static file handler
	mux.GetHandler("/:rest", http.FileServer(http.FS(publicFiles)))

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
