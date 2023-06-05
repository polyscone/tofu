package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func SignOut(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Post("/sign-out", signOutPost(h), "account.sign_out.post")
}

func signOutPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		_, err := h.RenewSession(ctx)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.Sessions.Destroy(r.Context())

		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}

}
