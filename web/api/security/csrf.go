package security

import (
	"net/http"

	"github.com/polyscone/tofu/pkg/http/router"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/httputil"
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
