package web

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/cache"
	"github.com/polyscone/tofu/dev"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/fstack"
	"github.com/polyscone/tofu/web/handler"
)

//go:embed "all:ui/public"
var files embed.FS

const publicDir = "ui/public"

var publicFiles = fstack.New(dev.RelDirFS(publicDir), errsx.Must(fs.Sub(files, publicDir)))

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
			api := NewAPIRouter(h)

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, app.BasePath+"/api/") {
					api.ServeHTTP(w, r)

					return
				}

				pwa.ServeHTTP(w, r)
			})
		}

		return mux
	})
}

type fsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	buf        *bytes.Buffer
}

func (w *fsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *fsResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *fsResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
