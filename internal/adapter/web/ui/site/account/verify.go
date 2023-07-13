package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Verify(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/verify", func(mux *router.ServeMux) {
		mux.Get("/", h.HTML.Handler("site/account/verify/form"), "account.verify")
		mux.Post("/", verifyPost(h), "account.verify.post")
	})
}

func verifyPost(h *ui.Handler) http.HandlerFunc {
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
			http.Redirect(w, r, h.Path("account.verify"), http.StatusSeeOther)

			return
		}

		email, err := h.Repo.Web.FindVerificationTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find verification token email", err, "site/error", nil)

			return
		}

		err = h.Svc.Account.VerifyUser(ctx, email, input.Password, input.PasswordCheck)
		if err != nil {
			h.HTML.ErrorView(w, r, "verify user", err, "site/account/verify/form", nil)

			return
		}

		err = h.Repo.Web.ConsumeVerificationToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume verification token", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Your account has been successfully verified.")

		signInWithPassword(ctx, h, w, r, email, input.Password)
	}
}
