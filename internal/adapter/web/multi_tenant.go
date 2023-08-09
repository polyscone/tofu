package web

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/slogger"
)

var ErrTenantNotFound = errors.New("not found")

type NewTenantFunc func(hostname string) (*handler.Tenant, error)

type MultiTenantHandler struct {
	behindSecureProxy bool
	logger            *slog.Logger
	logStyle          slogger.Style
	newTenant         NewTenantFunc
	muxesMu           sync.RWMutex
	muxes             map[string]http.Handler
}

func NewMultiTenantHandler(behindSecureProxy bool, logger *slog.Logger, logStyle slogger.Style, newTenant NewTenantFunc) *MultiTenantHandler {
	return &MultiTenantHandler{
		behindSecureProxy: behindSecureProxy,
		logger:            logger,
		logStyle:          logStyle,
		newTenant:         newTenant,
		muxes:             make(map[string]http.Handler),
	}
}

func (h *MultiTenantHandler) mux(r *http.Request) (http.Handler, error) {
	hostname, port, _ := strings.Cut(r.Host, ":")

	h.muxesMu.RLock()

	if mux, ok := h.muxes[hostname]; ok {
		h.muxesMu.RUnlock()

		return mux, nil
	}

	h.muxesMu.RUnlock()

	h.muxesMu.Lock()
	defer h.muxesMu.Unlock()

	if mux, ok := h.muxes[hostname]; ok {
		return mux, nil
	}

	tenant, err := h.newTenant(hostname)
	if err != nil {
		return nil, fmt.Errorf("new tenant: %w", err)
	}

	logger, err := slogger.New(h.logStyle, nil)
	if err != nil {
		return nil, fmt.Errorf("new logger: %w", err)
	}
	tenant.Log = logger

	if r.TLS != nil || h.behindSecureProxy {
		tenant.Scheme = "https"
	} else {
		tenant.Scheme = "http"
	}

	tenant.Host = r.Host
	tenant.Hostname = hostname
	tenant.Port = port

	mux := NewRouter(tenant)

	h.muxes[hostname] = mux

	return mux, nil
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
