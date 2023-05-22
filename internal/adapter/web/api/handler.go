package api

import (
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/api/handler/account"
	"github.com/polyscone/tofu/internal/adapter/web/api/handler/security"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

func NewHandler(tenant *handler.Tenant) http.Handler {
	mux := router.NewServeMux()
	svc := handler.NewServices(mux, tenant, nil)

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		svc.ErrorJSON(w, r, errors.Tracef(err))
	}

	// Middleware
	mux.Use(middleware.Recover(errorHandler))
	mux.Use(middleware.Timeout(5*time.Second, errorHandler))
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

	// Security
	mux.Prefix("/security", func(mux *router.ServeMux) {
		security.CSRF(svc, mux)
	})

	// Account
	mux.Prefix("/account", func(mux *router.ServeMux) {
		account.Activate(svc, mux)
		account.ChangePassword(svc, mux)
		account.ResetPassword(svc, mux)
		account.Login(svc, mux)
		account.Logout(svc, mux)
		account.Register(svc, mux)
		account.TOTP(svc, mux)
	})

	// Generic not found handler
	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		svc.ErrorJSON(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL))
	})

	// Generic method not allowed handler
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		svc.ErrorJSON(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL))
	})

	return mux
}
