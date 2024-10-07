package web

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/cache"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/handler"
)

// HandlerTimeout should be used as the value in all timeout middleware, and as the
// base value to calculate http.Server timeouts from.
const HandlerTimeout = 5 * time.Second

var muxes = cache.New[string, *http.ServeMux]()

func NewRouter(tenant *handler.Tenant) http.Handler {
	key := tenant.Key + "." + tenant.Kind

	return muxes.LoadOrStore(key, func() *http.ServeMux {
		mux := http.NewServeMux()
		h := handler.New(tenant)

		switch tenant.Kind {
		case "site":
			mux.Handle("/", NewSiteRouter(h))

		case "pwa":
			pwa := NewPWARouter(h)
			apiV1 := NewAPIRouterV1(h)

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, app.BasePath+"/api/") {
					apiV1.ServeHTTP(w, r)

					return
				}

				pwa.ServeHTTP(w, r)
			})
		}

		return mux
	})
}

type FileServerErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

func newFileServer(basePath string, mux *router.ServeMux, renderer *handler.Renderer, errorHandler FileServerErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if allowedMethods, notAllowed := httpx.MethodNotAllowed(mux, r); notAllowed {
			w.Header().Set("allow", strings.Join(allowedMethods, ", "))

			errorHandler(w, r, httpx.ErrMethodNotAllowed)

			return
		}

		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		if basePath != "" {
			upath = strings.TrimPrefix(upath, basePath)
		}
		upath = path.Clean(upath)

		if location, ok := renderer.AssetTagLocation(upath); ok {
			maxAge := int((365 * 24 * time.Hour).Seconds())
			cacheControl := fmt.Sprintf("public, max-age=%v, immutable", maxAge)
			w.Header().Set("cache-control", cacheControl)

			upath = location
		}

		name, modtime, b, err := renderer.Asset(r, upath)
		if err != nil {
			errorHandler(w, r, err)

			return
		}

		http.ServeContent(w, r, name, modtime, bytes.NewReader(b))
	}
}
