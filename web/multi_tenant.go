package web

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/pwa"
	"github.com/polyscone/tofu/web/site"
)

// HandlerTimeout should be used as the value in all timeout middleware, and as the
// base value to calculate http.Server timeouts from.
const HandlerTimeout = 5 * time.Second

var muxes = cache.New[string, *http.ServeMux]()

var ErrTenantNotFound = errors.New("not found")

type NewTenantFunc func(hostname string) (*handler.Tenant, error)

type MultiTenantHandler struct {
	logger    *slog.Logger
	newTenant NewTenantFunc
	config    handler.Config
	muxes     *cache.Cache[string, http.Handler]
}

func NewMultiTenantHandler(logger *slog.Logger, newTenant NewTenantFunc, config handler.Config) *MultiTenantHandler {
	return &MultiTenantHandler{
		logger:    logger,
		newTenant: newTenant,
		config:    config,
		muxes:     cache.New[string, http.Handler](),
	}
}

func (mh *MultiTenantHandler) mux(r *http.Request) (http.Handler, error) {
	return mh.muxes.LoadOrMaybeStore(r.Host, func() (http.Handler, error) {
		tenant, err := mh.newTenant(r.Host)
		if err != nil {
			return nil, fmt.Errorf("new tenant: %w", err)
		}
		if tenant.Logger == nil {
			panic("tenant logger must be set")
		}

		tenant.Scheme = "http"
		if httpx.IsTLS(r) {
			tenant.Scheme = "https"
		}

		tenant.Host = r.Host

		key := tenant.Key + "." + tenant.Kind
		return muxes.LoadOrMaybeStore(key, func() (*http.ServeMux, error) {
			mux := http.NewServeMux()
			h, err := handler.New(tenant)
			if err != nil {
				return nil, fmt.Errorf("new tenant: %w", err)
			}

			var router http.Handler
			switch tenant.Kind {
			case "site":
				router = site.NewRouter(h, HandlerTimeout, mh.config.Site)

			case "pwa":
				router = pwa.NewRouter(h, HandlerTimeout, mh.config.PWA)
			}

			if router != nil {
				apiV1 := api.NewRouterV1(h, HandlerTimeout, mh.config.APIv1)

				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, app.BasePath+"/api/") {
						apiV1.ServeHTTP(w, r)

						return
					}

					router.ServeHTTP(w, r)
				})
			}

			return mux, nil
		})
	})
}

func (mh *MultiTenantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux, err := mh.mux(r)
	if err != nil {
		mh.logger.Error("serve HTTP", "error", err, "url", r.URL.String())

		if errors.Is(err, ErrTenantNotFound) {
			http.Error(w, "Site not served on this interface", http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	mux.ServeHTTP(w, r)
}
