package meta

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/api/v1/ui"
)

func RegisterHealthHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("HEAD /api/v1/meta/health", healthGet(h))
	mux.HandleFunc("GET /api/v1/meta/health", healthGet(h))
}

func healthGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cache-control", "no-cache")

		if r.Method == http.MethodGet {
			h.RawJSON(w, r, http.StatusOK, []byte(`{"status":"available"}`))
		}
	}
}
