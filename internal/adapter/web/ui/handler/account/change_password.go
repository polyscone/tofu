package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

func ChangePassword(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/change-password", func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)
		mux.Before(func(w http.ResponseWriter, r *http.Request) bool {
			ctx := r.Context()
			user := h.User(ctx)

			if len(user.HashedPassword) == 0 {
				http.Redirect(w, r, h.Path("account.choose_password"), http.StatusSeeOther)

				return false
			}

			return true
		})

		mux.Get("/", changePasswordGet(h), "account.change_password")
		mux.Post("/", changePasswordPost(h), "account.change_password.post")
	})

	// Redirect to help password managers find the change password page
	mux.Redirect(http.MethodGet, "/.well-known/change-password", h.Path("account.change_password"), http.StatusSeeOther)
}

func changePasswordGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/change_password/form", nil)
	}
}

func changePasswordPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OldPassword      string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		log := h.Logger(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		err := h.Account.ChangePassword(ctx,
			passport.Account,
			user.ID,
			input.OldPassword,
			input.NewPassword,
			input.NewPasswordCheck,
		)
		if err != nil {
			h.ErrorView(w, r, "change password", err, "account/change_password/form", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.ErrorView(w, r, "renew session", err, "error", nil)

			return
		}

		knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(input.NewPassword))
		if err != nil {
			log.Error("known password breach count", "error", err)

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
