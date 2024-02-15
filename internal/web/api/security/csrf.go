package security

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/web/api"
	"github.com/polyscone/tofu/internal/web/httputil"
)

func RegisterCSRFHandlers(h *api.Handler, mux *router.ServeMux) {
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
