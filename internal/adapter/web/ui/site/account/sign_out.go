package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func signOutRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Post("/sign-out", signOutPost(h), "account.sign_out.post")
}

func signOutPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		_, err := h.RenewSession(ctx)
		if err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "site/error", nil)

			return
		}

		h.Sessions.Destroy(r.Context())

		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}

}
