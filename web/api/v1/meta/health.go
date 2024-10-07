package meta

import (
	"net/http"

	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/api"
)

func RegisterHealthHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("HEAD /api/v1/meta/health", healthGet(h))
	mux.HandleFunc("GET /api/v1/meta/health", healthGet(h))
}

func healthGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cache-control", "no-cache")

		if r.Method == http.MethodGet {
			h.RawJSON(w, r, http.StatusOK, []byte(`{"status":"available"}`))
		}
	}
}
