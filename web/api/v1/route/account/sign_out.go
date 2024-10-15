package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/api/v1/ui"
)

func RegisterSignOutHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /api/v1/account/sign-out", signOutPost(h))
}

func signOutPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		_, err := h.RenewSession(ctx)
		if err != nil {
			h.ErrorJSON(w, r, "renew session", err)

			return
		}

		h.Session.Destroy(r.Context())

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}
