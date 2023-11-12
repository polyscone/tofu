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

type handlerCache struct {
	api  http.Handler
	pwa  http.Handler
	site http.Handler
}

var (
	muxes    = cache.New[string, *http.ServeMux]()
	handlers = make(map[string]*handlerCache)
)

func NewRouter(tenant *handler.Tenant) http.Handler {
	key := tenant.Key + "." + tenant.Kind

	return muxes.LoadOrStore(key, func() *http.ServeMux {
		mux := http.NewServeMux()
		h := handler.New(tenant)

		hc, ok := handlers[tenant.Key]
		if !ok {
			hc = &handlerCache{
				api:  NewAPIRouter(h),
				pwa:  NewPWARouter(h),
				site: NewSiteRouter(h),
			}

			handlers[tenant.Key] = hc
		}

		switch tenant.Kind {
		case "site":
			mux.Handle("/", hc.site)

		case "pwa":
			mux.Handle("/", hc.pwa)
		}

		mux.Handle("/api/v1/", http.StripPrefix("/api/v1", hc.api))

		return mux
	})
}
