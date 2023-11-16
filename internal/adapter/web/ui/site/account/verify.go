package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func verifyRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/verify", func(mux *router.ServeMux) {
		mux.Get("/", h.HTML.Handler("site/account/verify/form"), "account.verify")
		mux.Post("/", verifyPost(h), "account.verify.post")

		mux.Get("/success", h.HTML.Handler("site/account/verify/success"), "account.verify.success")
	})
}

func verifyPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token         string `form:"token"`
			Password      string `form:"password"`
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.verify"), http.StatusSeeOther)

			return
		}

		email, err := h.Repo.Web.FindEmailVerificationTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find verification token email", err, "site/account/verify/form", nil)

			return
		}

		behaviour := account.VerifyUserActivate
		if !config.SignUpAutoActivateEnabled {
			behaviour = account.VerifyUserOnly
		}

		err = h.Svc.Account.VerifyUser(ctx, email, input.Password, input.PasswordCheck, behaviour)
		if err != nil {
			h.HTML.ErrorView(w, r, "verify user", err, "site/account/verify/form", nil)

			return
		}

		err = h.Repo.Web.ConsumeEmailVerificationToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume verification token", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Your account has been successfully verified.")

		if behaviour == account.VerifyUserActivate {
			signInWithPassword(ctx, h, w, r, email, input.Password)
		} else {
			http.Redirect(w, r, h.Path("account.verify.success"), http.StatusSeeOther)
		}
	}
}
