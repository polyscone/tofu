package meta

import (
	"net/http"

	"github.com/polyscone/tofu/http/router"
	"github.com/polyscone/tofu/web/api"
)

func RegisterHealthHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("HEAD /meta/health", healthGet(h))
	mux.HandleFunc("GET /meta/health", healthGet(h))
}

func healthGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.Header().Set("cache-control", "no-cache")

		if r.Method == http.MethodGet {
			w.Write([]byte(`{"status":"available"}`))
		}
	}
}
