package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func ResetPassword(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/reset-password", resetPasswordGet(svc), "account.reset_password")
	mux.Post("/reset-password", resetPasswordPost(svc), "account.reset_password.post")
	mux.Put("/reset-password", resetPasswordPut(svc), "account.reset_password.put")
}

func resetPasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/reset_password", nil)
	}
}

func resetPasswordPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if svc.ErrorView(w, r, errors.Tracef(httputil.DecodeForm(&input, r)), "error", nil) {
			return
		}

		if _, err := text.NewEmail(input.Email); err != nil {
			svc.ErrorViewFunc(w, r, errors.Tracef(err), "account/reset_password", func(data *handler.ViewData) {
				data.Errors = errors.Map{"email": err}
			})

			return
		}

		svc.Broker.Dispatch(handler.ResetPasswordRequested{
			Email: input.Email,
		})

		http.Redirect(w, r, svc.Path("account.reset_password")+"?status=email-sent", http.StatusSeeOther)
	}
}

func resetPasswordPut(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		email, err := svc.Repo.Web.FindResetPasswordTokenEmail(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		passport, err := svc.PassportByEmail(ctx, email)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = svc.Account.ResetPassword(ctx, passport, passport.UserID(), input.NewPassword, input.NewPasswordCheck)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/reset_password", nil) {
			return
		}

		err = svc.Repo.Web.ConsumeResetPasswordToken(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.Flash, "Your password has been successfully changed.")

		loginWithPassword(ctx, svc, w, r, email, input.NewPassword)
	}
}
