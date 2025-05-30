package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterSignOutHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/sign-out", signOutGet(h))
	mux.HandleFunc("POST /account/sign-out", signOutPost(h), "account.sign_out.post")
}

func signOutGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}
}

func signOutPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "error", nil)

			return
		}

		h.Session.Destroy(r.Context())

		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}

}
