package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

func ChangePassword(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/change-password", func(mux *router.ServeMux) {
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

		mux.Get("/", h.HTML.Handler("site/account/change_password/form"), "account.change_password")
		mux.Post("/", changePasswordPost(h), "account.change_password.post")
	})

	// Redirect to help password managers find the change password page
	mux.Redirect(http.MethodGet, "/.well-known/change-password", h.Path("account.change_password"), http.StatusSeeOther)
}

func changePasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OldPassword      string `form:"old-password"`
			NewPassword      string `form:"new-password"`
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		logger := h.Logger(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		err := h.Svc.Account.ChangePassword(ctx,
			passport.Account,
			user.ID,
			input.OldPassword,
			input.NewPassword,
			input.NewPasswordCheck,
		)
		if err != nil {
			h.HTML.ErrorView(w, r, "change password", err, "site/account/change_password/form", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "site/error", nil)

			return
		}

		knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(input.NewPassword))
		if err != nil {
			logger.Error("known password breach count", "error", err)

			h.Sessions.Delete(ctx, sess.KnownPasswordBreachCount)
		} else {
			if knownBreachCount > 0 {
				h.Sessions.Set(ctx, sess.KnownPasswordBreachCount, knownBreachCount)
			} else {
				h.Sessions.Delete(ctx, sess.KnownPasswordBreachCount)
			}
		}

		h.AddFlashf(ctx, "Your password has been successfully changed.")

		var redirect string
		if r := h.Sessions.PopString(ctx, sess.Redirect); r != "" {
			redirect = r
		} else {
			redirect = h.Path("account.change_password")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
