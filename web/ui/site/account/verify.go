package account

import (
	"net/http"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/sess"
	"github.com/polyscone/tofu/web/ui"
)

func RegisterVerifyHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/verify", verifyGet(h), "account.verify")
	mux.HandleFunc("POST /account/verify", verifyPost(h), "account.verify.post")

	mux.HandleFunc("GET /account/verify/success", h.HTML.HandlerFunc("site/account/verify/success"), "account.verify.success")
}

func verifyGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.HTML.View(w, r, http.StatusOK, "site/account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/verify/form", nil)
	}
}

func verifyPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token         string `form:"token"`
			Password      string `form:"password"`
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.verify"), http.StatusSeeOther)

			return
		}

		ctx := r.Context()

		email, behaviour, err := auth.Verify(ctx, h.Handler, w, r, input.Token, input.Password, input.PasswordCheck)
		if err != nil {
			h.HTML.ErrorView(w, r, "verify sign up", err, "site/account/verify/form", nil)

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
