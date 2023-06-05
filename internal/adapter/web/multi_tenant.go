package web

import (
	"net/http"
	"strings"
	"sync"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

var ErrTenantNotFound = errors.New("tenant not found")

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

	mux := http.NewServeMux()

	mux.Handle("/", ui.NewRouter(tenant))

	h.handlers[hostname] = mux

	return mux, nil
}

func (h *MultiTenantHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, err := h.handler(r)
	if err != nil {
		httputil.LogError(r, errors.Tracef(err))

		if errors.Is(err, ErrTenantNotFound) {
			http.Error(w, "Site not served on this interface", http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	handler.ServeHTTP(w, r)
}
