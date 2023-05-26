package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Activate(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/activate", activateGet(svc), "account.activate")
	mux.Post("/activate", activatePost(svc), "account.activate.post")
}

func activateGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/activate", nil)
	}
}

func activatePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		if input.Token == "" {
			http.Redirect(w, r, svc.Path("account.activate"), http.StatusSeeOther)

			return
		}

		email, err := svc.Repo.Web.FindActivationTokenEmail(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = svc.Account.ActivateUser(ctx, email)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/activate", nil) {
			return
		}

		err = svc.Repo.Web.ConsumeActivationToken(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account.activate")+"?status=success", http.StatusSeeOther)
	}
}
