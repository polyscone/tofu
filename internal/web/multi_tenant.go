package web

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/polyscone/tofu/internal/pkg/cache"
	"github.com/polyscone/tofu/internal/web/handler"
)

var ErrTenantNotFound = errors.New("not found")

type NewTenantFunc func(hostname string) (*handler.Tenant, error)

type MultiTenantHandler struct {
	logger            *slog.Logger
	behindSecureProxy bool
	newTenant         NewTenantFunc
	muxesMu           sync.RWMutex
	muxes             *cache.Cache[string, http.Handler]
}

func NewMultiTenantHandler(logger *slog.Logger, behindSecureProxy bool, newTenant NewTenantFunc) *MultiTenantHandler {
	return &MultiTenantHandler{
		logger:            logger,
		behindSecureProxy: behindSecureProxy,
		newTenant:         newTenant,
		muxes:             cache.New[string, http.Handler](),
	}
}

func (h *MultiTenantHandler) mux(r *http.Request) (http.Handler, error) {
	return h.muxes.LoadOrMaybeStore(r.Host, func() (http.Handler, error) {
		tenant, err := h.newTenant(r.Host)
		if err != nil {
			return nil, fmt.Errorf("new tenant: %w", err)
		}
		if tenant.Logger == nil {
			panic("tenant logger must be set")
		}

		if r.TLS != nil || h.behindSecureProxy {
			tenant.Scheme = "https"
		} else {
			tenant.Scheme = "http"
		}

		tenant.Host = r.Host

		return NewRouter(tenant), nil
	})
}

func (h *MultiTenantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux, err := h.mux(r)
	if err != nil {
		h.logger.Error("serve HTTP", "error", err, "url", r.URL.String())

		if errors.Is(err, ErrTenantNotFound) {
			http.Error(w, "Site not served on this interface", http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	mux.ServeHTTP(w, r)
}
