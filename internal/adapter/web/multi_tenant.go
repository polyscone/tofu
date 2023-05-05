package web

import (
	"net/http"
	"strings"
	"sync"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

type NewTenantFunc func(hostname string) (*handler.Tenant, error)

type MultiTenantHandler struct {
	newTenant  NewTenantFunc
	handlersMu sync.RWMutex
	handlers   map[string]http.Handler
}

func NewMultiTenantHandler(newTenant NewTenantFunc) *MultiTenantHandler {
	return &MultiTenantHandler{
		newTenant: newTenant,
		handlers:  make(map[string]http.Handler),
	}
}

func (h *MultiTenantHandler) handler(r *http.Request) (http.Handler, error) {
	hostname, port, _ := strings.Cut(r.Host, ":")

	h.handlersMu.RLock()

	if handler, ok := h.handlers[hostname]; ok {
		h.handlersMu.RUnlock()

		return handler, nil
	}

	h.handlersMu.RUnlock()

	h.handlersMu.Lock()
	defer h.handlersMu.Unlock()

	if handler, ok := h.handlers[hostname]; ok {
		return handler, nil
	}

	tenant, err := h.newTenant(hostname)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if r.TLS != nil {
		tenant.Scheme = "https"
	} else {
		tenant.Scheme = "http"
	}

	tenant.Host = r.Host
	tenant.Hostname = hostname
	tenant.Port = port

	mux := router.NewServeMux()

	mux.AnyHandler("/api/v1/:rest", http.StripPrefix("/api/v1", api.NewHandler(tenant)))
	mux.AnyHandler("/:rest", ui.NewHandler(tenant))

	h.handlers[hostname] = mux

	return mux, nil
}

func (h *MultiTenantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, err := h.handler(r)
	if err != nil {
		httputil.LogError(r, errors.Tracef(err))

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	handler.ServeHTTP(w, r)
}
