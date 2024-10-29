package account

import (
	"net/http"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterVerifyHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/verify", verifyGet(h), "account.verify")
	mux.HandleFunc("POST /account/verify", verifyPost(h), "account.verify.post")

	mux.HandleFunc("GET /account/verify/success", verifySuccessGet(h), "account.verify.success")
}

func verifyGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Session.IsSignedIn(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/verify/form", nil)
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
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.verify"), http.StatusSeeOther)

			return
		}

		ctx := r.Context()

		email, behaviour, err := auth.Verify(ctx, h.Handler, w, r, input.Token, input.Password, input.PasswordCheck)
		if err != nil {
			h.HTML.ErrorView(w, r, "verify sign up", err, "account/verify/form", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.verify.flash.success"))

		if behaviour == account.VerifyUserActivate {
			signInWithPassword(ctx, h, w, r, email, input.Password)
		} else {
			http.Redirect(w, r, h.Path("account.verify.success"), http.StatusSeeOther)
		}
	}
}

func verifySuccessGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/verify/success", nil)
	}
}
