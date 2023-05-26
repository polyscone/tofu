package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

func ChangePassword(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Get("/change-password", changePasswordGet(svc), "account.change_password")
	mux.Put("/change-password", changePasswordPut(svc), "account.change_password.put")

	// Redirect to help password managers find the change password page
	mux.Redirect(http.MethodGet, "/.well-known/change-password", svc.Path("account.change_password"), http.StatusSeeOther)

	guard.ProtectPrefix(svc.Path("account.change_password"))

	svc.SetViewVars("account/change_password", handler.Vars{
		"NewPasswordKnownBreachCount": 0,
	})
}

func changePasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/change_password", nil)
	}
}

func changePasswordPut(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OldPassword      string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
			InsecurePassword string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		knownBreachCount, err := pwned.PasswordKnownBreachCount(ctx, []byte(input.NewPassword))
		if err != nil {
			httputil.LogError(r, err)
		}

		if input.NewPassword == input.InsecurePassword {
			if knownBreachCount > 0 {
				svc.Sessions.Set(ctx, sess.PasswordKnownBreachCount, knownBreachCount)
			} else {
				svc.Sessions.Delete(ctx, sess.PasswordKnownBreachCount)
			}
		} else if knownBreachCount > 0 {
			svc.View(w, r, http.StatusOK, "account/change_password", handler.Vars{
				"NewPasswordKnownBreachCount": knownBreachCount,
			})

			return
		}

		err = svc.Account.ChangePassword(ctx,
			passport,
			passport.UserID(),
			input.OldPassword,
			input.NewPassword,
			input.NewPasswordCheck,
		)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/change_password", nil) {
			return
		}

		_, err = svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		var redirect string
		if r := svc.Sessions.PopString(ctx, sess.Redirect); r != "" {
			svc.Sessions.Set(ctx, sess.Flash, "Your password has been successfully changed.")

			redirect = r
		} else {
			redirect = svc.Path("account.change_password") + "?status=success"
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
