package account

import (
	"net/http"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/internal/password/pwned"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterChoosePasswordHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Named("account.choose_password.section", "/account/choose-password")

	mux.Group(func(mux *router.ServeMux) {
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				user := h.User(ctx)

				if h.Session.IsSignedIn(ctx) && len(user.HashedPassword) > 0 {
					http.Redirect(w, r, h.Path("account.change_password"), http.StatusSeeOther)

					return
				}

				next(w, r)
			}
		})

		mux.HandleFunc("GET /account/choose-password", choosePasswordGet(h), "account.choose_password")
		mux.HandleFunc("POST /account/choose-password", choosePasswordPost(h), "account.choose_password.post")
	})
}

func choosePasswordGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.RequireSignIn(w, r) {
			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/choose_password/form", nil)
	}
}

func choosePasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.RequireSignIn(w, r) {
			return
		}

		var input struct {
			NewPassword      string `form:"new-password"`
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		logger := h.Logger(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, err := h.Svc.Account.ChoosePassword(ctx, passport.Account, account.ChoosePasswordInput{
			UserID:           user.ID,
			NewPassword:      input.NewPassword,
			NewPasswordCheck: input.NewPasswordCheck,
		})
		if err != nil {
			h.HTML.ErrorView(w, r, "choose password", err, h.Session.LastView(ctx), nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "error", nil)

			return
		}

		knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(input.NewPassword))
		if err != nil {
			logger.Error("known password breach count", "error", err)

			h.Session.DeleteKnownPasswordBreachCount(ctx)
		} else {
			if knownBreachCount > 0 {
				h.Session.SetKnownPasswordBreachCount(ctx, knownBreachCount)
			} else {
				h.Session.DeleteKnownPasswordBreachCount(ctx)
			}
		}

		h.AddFlashf(ctx, i18n.M("site.account.choose_password.flash.password_chosen"))

		var redirect string
		if r := h.Session.PopRedirect(ctx); r != "" {
			redirect = r
		} else {
			redirect = h.Path("account.change_password")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
