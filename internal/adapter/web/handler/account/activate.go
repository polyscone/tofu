package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Activate(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/activate", func(mux *router.ServeMux) {
		mux.Get("/", h.HTML.Handler("site/account/activate/form"), "account.activate")
		mux.Post("/", activatePost(h), "account.activate.post")
	})
}

func activatePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token         string
			Password      string
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.activate"), http.StatusSeeOther)

			return
		}

		email, err := h.Repo.Web.FindActivationTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find activation token email", err, "site/error", nil)

			return
		}

		err = h.Account.ActivateUser(ctx, email, input.Password, input.PasswordCheck)
		if err != nil {
			h.HTML.ErrorView(w, r, "activate user", err, "site/account/activate/form", nil)

			return
		}

		err = h.Repo.Web.ConsumeActivationToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume activation token", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Your account has been successfully activated.")

		signInWithPassword(ctx, h, w, r, email, input.Password)
	}
}
