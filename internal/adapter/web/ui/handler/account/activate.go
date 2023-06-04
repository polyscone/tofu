package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Activate(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/activate", func(mux *router.ServeMux) {
		mux.Get("/", activateGet(svc), "account.activate")
		mux.Post("/", activatePost(svc), "account.activate.post")
	})
}

func activateGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/activate/form", nil)
	}
}

func activatePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token         string
			Password      string
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
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

		err = svc.Account.ActivateUser(ctx, email, input.Password, input.PasswordCheck)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/activate/form", nil) {
			return
		}

		err = svc.Repo.Web.ConsumeActivationToken(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.AddFlashf(ctx, "Your account has been successfully activated.")

		signInWithPassword(ctx, svc, w, r, email, input.Password)
	}
}
