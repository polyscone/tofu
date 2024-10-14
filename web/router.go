package web

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
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

		var router http.Handler
		switch tenant.Kind {
		case "site":
			router = NewSiteRouter(h)

		case "pwa":
			router = NewPWARouter(h)
		}

		if router != nil {
			apiV1 := NewAPIRouterV1(h)

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, app.BasePath+"/api/") {
					apiV1.ServeHTTP(w, r)

					return
				}

				router.ServeHTTP(w, r)
			})
		}

		return mux
	})
}

type FileServerErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

func newFileServer(mux *router.ServeMux, renderer *handler.Renderer, errorHandler FileServerErrorHandler) http.HandlerFunc {
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
		upath = path.Clean(upath)

		tagged := upath
		if q := r.URL.RawQuery; q != "" {
			tagged += "?" + q
		}
		if asset, ok := renderer.FindAssetByTagged(tagged); ok {
			maxAge := int((365 * 24 * time.Hour).Seconds())
			cacheControl := fmt.Sprintf("public, max-age=%v, immutable", maxAge)
			w.Header().Set("cache-control", cacheControl)

			upath = asset
		}

		name, modtime, b, err := renderer.Asset(r, nil, upath)
		if err != nil {
			errorHandler(w, r, err)

			return
		}

		http.ServeContent(w, r, name, modtime, bytes.NewReader(b))
	}
}
