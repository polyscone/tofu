package web

import (
	"embed"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
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
	site http.Handler
	pwa  http.Handler
	api  http.Handler
}

var muxes = struct {
	mu       sync.RWMutex
	data     map[string]*http.ServeMux
	handlers map[string]*handlerCache
}{
	data:     make(map[string]*http.ServeMux),
	handlers: make(map[string]*handlerCache),
}

func NewRouter(tenant *handler.Tenant) http.Handler {
	key := tenant.Key + "." + tenant.Kind

	muxes.mu.RLock()

	if mux, ok := muxes.data[key]; ok {
		muxes.mu.RUnlock()

		return mux
	}

	muxes.mu.RUnlock()

	muxes.mu.Lock()
	defer muxes.mu.Unlock()

	if mux, ok := muxes.data[key]; ok {
		return mux
	}

	mux := http.NewServeMux()
	h := handler.New(tenant)

	handlers, ok := muxes.handlers[tenant.Key]
	if !ok {
		handlers = &handlerCache{
			site: NewSiteRouter(h),
			pwa:  NewPWARouter(h),
			api:  NewAPIRouter(h),
		}

		muxes.handlers[tenant.Key] = handlers
	}

	switch tenant.Kind {
	case "site":
		mux.Handle("/", handlers.site)

	case "pwa":
		mux.Handle("/", handlers.pwa)
	}

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", handlers.api))

	muxes.data[key] = mux

	return mux
}
