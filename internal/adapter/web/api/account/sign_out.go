package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func signOutRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.Post("/sign-out", signOutPost(h))
}

func signOutPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		_, err := h.RenewSession(ctx)
		if err != nil {
			h.ErrorJSON(w, r, "renew session", err)

			return
		}

		h.Sessions.Destroy(r.Context())

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}
