package account

import (
	"net/http"

	"github.com/polyscone/tofu/http/middleware"
	"github.com/polyscone/tofu/http/router"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/httputil"
)

func RegisterSignOutHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /account/sign-out", signOutPost(h))
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
