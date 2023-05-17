package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

func ResetPassword(svc *handler.Services, mux *router.ServeMux, tokens token.Repo) {
	mux.Get("/reset-password", resetPasswordGet(svc), "account.reset_password")
	mux.Post("/reset-password", resetPasswordPost(svc, tokens), "account.reset_password.post")
	mux.Put("/reset-password", resetPasswordPut(svc, tokens), "account.reset_password.put")
}

func resetPasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/reset_password", nil)
	}
}

func resetPasswordPost(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if svc.ErrorView(w, r, errors.Tracef(httputil.DecodeForm(r, &input)), "error", nil) {
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

func resetPasswordPut(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(r, &input)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		email, err := tokens.FindResetPasswordTokenEmail(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		passport, err := svc.LimitedPassportByEmail(ctx, email.String())
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		cmd := account.ResetPassword{
			Guard:            passport,
			UserID:           passport.UserID(),
			NewPassword:      input.NewPassword,
			NewPasswordCheck: input.NewPasswordCheck,
		}
		err = cmd.Validate(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/reset_password", nil) {
			return
		}

		// Only consume after manual command validation, but before execution
		// This way the token will only be consumed once we know there aren't any
		// input validation or authorisation errors
		err = tokens.ConsumeResetPasswordToken(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = cmd.Execute(ctx, svc.Bus)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/reset_password", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.Flash, "Your password has been successfully changed.")

		loginWithPassword(ctx, svc, w, r, email.String(), input.NewPassword)
	}
}
