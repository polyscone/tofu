package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/password/pwned"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterChangePasswordHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				user := h.User(ctx)

				if len(user.HashedPassword) == 0 {
					http.Redirect(w, r, h.Path("account.choose_password"), http.StatusSeeOther)

					return
				}

				next(w, r)
			}
		})

		mux.HandleFunc("GET /account/change-password", changePasswordGet(h), "account.change_password")
		mux.HandleFunc("POST /account/change-password", changePasswordPost(h), "account.change_password.post")
	})

	// Redirect to help password managers find the change password page
	mux.Handle("/.well-known/change-password", http.RedirectHandler(h.Path("account.change_password"), http.StatusSeeOther))
}

func changePasswordGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/change_password/form", nil)
	}
}

func changePasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OldPassword      string `form:"old-password"`
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

		_, err := h.Svc.Account.ChangePassword(ctx,
			passport.Account,
			user.ID,
			input.OldPassword,
			input.NewPassword,
			input.NewPasswordCheck,
		)
		if err != nil {
			h.HTML.ErrorView(w, r, "change password", err, "account/change_password/form", nil)

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

		h.AddFlashf(ctx, "Your password has been successfully changed.")

		var redirect string
		if r := h.Session.PopRedirect(ctx); r != "" {
			redirect = r
		} else {
			redirect = h.Path("account.change_password")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
