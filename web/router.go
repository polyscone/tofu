package web

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/polyscone/tofu/pkg/cache"
	"github.com/polyscone/tofu/pkg/dev"
	"github.com/polyscone/tofu/pkg/errsx"
	"github.com/polyscone/tofu/pkg/fstack"
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
			mux.Handle("/", NewPWARouter(h))
			mux.Handle("/api/v1/", http.StripPrefix("/api/v1", NewAPIRouter(h)))
		}

		return mux
	})
}
