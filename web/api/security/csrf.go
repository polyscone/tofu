package security

import (
	"net/http"

	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/api"
)

func RegisterCSRFHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /security/csrf", csrfGet(h))
}

func csrfGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.JSON(w, r, http.StatusOK, map[string]any{
			"csrfToken": httpx.MaskedCSRFToken(ctx),
		})
	}
}
