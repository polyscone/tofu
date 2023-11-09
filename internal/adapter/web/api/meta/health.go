package meta

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func healthRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.Prefix("/health", func(mux *router.ServeMux) {
		mux.Head("/", healthGet(h))
		mux.Get("/", healthGet(h))
	})
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
