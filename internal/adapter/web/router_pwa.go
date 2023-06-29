package web

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/size"
	"golang.org/x/exp/slices"
)

func NewPWARouter(base *handler.Handler) http.Handler {
	mux := router.NewServeMux()
	h := ui.NewHandler(base, mux, func() string {
		return "/sign-in"
	})

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
		Consume: func(r *http.Request) bool {
			whitelist := []string{".css", ".gif", ".ico", ".jpeg", ".jpg", ".js", ".png"}

			return !slices.ContainsFunc(whitelist, func(el string) bool {
				return strings.HasSuffix(r.URL.Path, el)
			})
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
				h.HTML.View(w, r, http.StatusOK, "pwa/root", nil)
			} else {
				ctx := r.Context()
				logger := h.Logger(ctx)

				logger.Error("static file", "error", err)

				http.Redirect(w, r, "/error", http.StatusSeeOther)
			}

			return
		}
		if stat.IsDir() {
			h.HTML.View(w, r, http.StatusOK, "pwa/root", nil)

			return
		}

		fileServer.ServeHTTP(w, r)
	}))

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

	return mux
}
