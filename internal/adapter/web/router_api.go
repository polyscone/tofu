package web

import (
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/api/account"
	"github.com/polyscone/tofu/internal/adapter/web/api/meta"
	"github.com/polyscone/tofu/internal/adapter/web/api/security"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

func NewAPIRouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()
	h := api.NewHandler(base)

	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			h.ErrorJSON(w, r, msg, err)
		}
	}

	mux.Use(middleware.Recover(errorHandler("recover middleware")))
	mux.Use(middleware.Timeout(5*time.Second, errorHandler("timeout middleware")))
	mux.Use(middleware.RemoveTrailingSlash)
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(middleware.Session(h.Sessions, &middleware.SessionConfig{
		Insecure:     h.Insecure,
		ErrorHandler: errorHandler("session middleware"),
	}))
	mux.Use(h.AttachContext)
	mux.Use(middleware.CSRF(&middleware.CSRFConfig{
		Insecure:     h.Insecure,
		ErrorHandler: errorHandler("CSRF middleware"),
	}))
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
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

	mux.Prefix("/account", func(mux *router.ServeMux) {
		account.SignIn(h, mux)
		account.SignOut(h, mux)
	})

	mux.Prefix("/security", func(mux *router.ServeMux) {
		security.CSRF(h, mux)
	})

	mux.Prefix("/meta", func(mux *router.ServeMux) {
		meta.Health(h, mux)
	})

	return mux
}
