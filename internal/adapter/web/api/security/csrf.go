package security

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func csrfRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /security/csrf", csrfGet(h))
}

func csrfGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.JSON(w, r, http.StatusOK, map[string]any{
			"csrfToken": httputil.MaskedCSRFToken(ctx),
		})
	}
}
