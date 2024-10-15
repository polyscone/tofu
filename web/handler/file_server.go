package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
)

type FileServerErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

func NewFileServer(mux *router.ServeMux, renderer *Renderer, errorHandler FileServerErrorHandler) http.HandlerFunc {
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
