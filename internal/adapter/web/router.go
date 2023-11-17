package web

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/cache"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/fstack"
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

		api := NewAPIRouter(h)
		pwa := NewPWARouter(h)
		site := NewSiteRouter(h)

		switch tenant.Kind {
		case "site":
			mux.Handle("/", site)

		case "pwa":
			mux.Handle("/", pwa)
		}

		mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api))

		return mux
	})
}
