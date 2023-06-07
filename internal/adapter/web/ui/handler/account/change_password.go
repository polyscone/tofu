package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

func ChangePassword(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Prefix("/change-password", func(mux *router.ServeMux) {
		guard.RequireSignIn()

		mux.Get("/", changePasswordGet(h), "account.change_password")
		mux.Post("/", changePasswordPost(h), "account.change_password.post")

		mux.Get("/success", changePasswordSuccessGet(h), "account.change_password.success")
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
			InsecurePassword string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		knownBreachCount, err := pwned.PasswordKnownBreachCount(ctx, []byte(input.NewPassword))
		if err != nil {
			httputil.LogError(r, err)
		}

		if input.NewPassword == input.InsecurePassword {
			if knownBreachCount > 0 {
				h.Sessions.Set(ctx, sess.PasswordKnownBreachCount, knownBreachCount)
			} else {
				h.Sessions.Delete(ctx, sess.PasswordKnownBreachCount)
			}
		} else if knownBreachCount > 0 {
			h.View(w, r, http.StatusOK, "account/change_password/form", handler.Vars{
				"NewPasswordKnownBreachCount": knownBreachCount,
			})

			return
		}

		err = h.Account.ChangePassword(ctx,
			passport,
			passport.UserID(),
			input.OldPassword,
			input.NewPassword,
			input.NewPasswordCheck,
		)
		if h.ErrorView(w, r, errors.Tracef(err), "account/change_password/form", nil) {
			return
		}

		_, err = h.RenewSession(ctx)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		var redirect string
		if r := h.Sessions.PopString(ctx, sess.Redirect); r != "" {
			h.AddFlashf(ctx, "Your password has been successfully changed.")

			redirect = r
		} else {
			redirect = h.Path("account.change_password.success")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}

func changePasswordSuccessGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/change_password/success", nil)
	}
}
