package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Activate(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/activate", func(mux *router.ServeMux) {
		mux.Get("/", activateGet(h), "account.activate")
		mux.Post("/", activatePost(h), "account.activate.post")
	})
}

func activateGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/activate/form", nil)
	}
}

func activatePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token         string
			Password      string
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.activate"), http.StatusSeeOther)

			return
		}

		email, err := h.Repo.Web.FindActivationTokenEmail(ctx, input.Token)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = h.Account.ActivateUser(ctx, email, input.Password, input.PasswordCheck)
		if h.ErrorView(w, r, errors.Tracef(err), "account/activate/form", nil) {
			return
		}

		err = h.Repo.Web.ConsumeActivationToken(ctx, input.Token)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.AddFlashf(ctx, "Your account has been successfully activated.")

		signInWithPassword(ctx, h, w, r, email, input.Password)
	}
}
