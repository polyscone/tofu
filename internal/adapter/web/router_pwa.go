package web

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
)

func setupPWARoutes(tenant *handler.Tenant, h *handler.Handler, mux *router.ServeMux) {
	errorHandler := func(msg string) middleware.ErrorHandler {
		return func(w http.ResponseWriter, r *http.Request, err error) {
			ctx := r.Context()
			logger := h.Logger(ctx)

			logger.Error(msg, "error", err)

			http.Redirect(w, r, "/error", http.StatusSeeOther)
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
		TrustedProxies: tenant.Proxies,
	}))
	mux.Use(middleware.Session(h.Sessions, &middleware.SessionConfig{
		Insecure:     tenant.Insecure,
		ErrorHandler: errorHandler("session middleware"),
	}))
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(middleware.CSRF(&middleware.CSRFConfig{
		Insecure:     tenant.Insecure,
		ErrorHandler: errorHandler("CSRF middleware"),
	}))
	mux.Use(middleware.Heartbeat("/meta/health"))
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))
	mux.Use(h.SetupMiddleware)

	setupPublicFileServerRoute(h, mux, func(w http.ResponseWriter, r *http.Request, err error) {
		switch {
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, fs.ErrInvalid):
			h.HTML.View(w, r, http.StatusOK, "pwa/root", nil)

		default:
			ctx := r.Context()
			logger := h.Logger(ctx)

			logger.Error("static file", "error", err)

			http.Redirect(w, r, "/error", http.StatusSeeOther)
		}
	})

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)

		logger.Error("handler", "error", httputil.ErrNotFound)

		http.Redirect(w, r, "/error", http.StatusSeeOther)
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)

		logger.Error("handler", "error", httputil.ErrMethodNotAllowed)

		http.Redirect(w, r, "/error", http.StatusSeeOther)
	})
}
