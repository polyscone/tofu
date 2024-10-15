package security

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/api/v1/ui"
)

func RegisterCSRFHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /api/v1/security/csrf", csrfGet(h))
}

func csrfGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.JSON(w, r, http.StatusOK, map[string]any{
			"csrfToken": httpx.MaskedCSRFToken(ctx),
		})
	}
}
