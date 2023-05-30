package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func SignOut(svc *handler.Services, mux *router.ServeMux) {
	mux.Post("/sign-out", signOutPost(svc), "account.sign_out.post")
}

func signOutPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		_, err := svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Destroy(r.Context())

		http.Redirect(w, r, svc.Path("account.sign_in"), http.StatusSeeOther)
	}

}
