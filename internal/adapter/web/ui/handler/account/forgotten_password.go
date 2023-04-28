package account

import (
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

func ForgottenPasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/forgotten_password", nil)
	}
}

func ForgottenPasswordPost(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if svc.RenderError(w, r, errors.Tracef(httputil.DecodeForm(r, &input)), "error", nil) {
			return
		}

		email, err := text.NewEmail(input.Email)
		stop := svc.RenderError(w, r, errors.Tracef(err), "account/forgotten_password", func(data *handler.Data) {
			data.Errors = errors.Map{"email": err}
		})
		if stop {
			return
		}

		ctx := r.Context()

		tok, err := tokens.AddResetPasswordToken(ctx, email, 2*time.Hour)
		if err != nil {
			logger.PrintError(err)

			return
		}

		msg := smtp.Msg{
			From:    "noreply@example.com",
			To:      []string{input.Email},
			Subject: "Reset your password",
			Plain:   "Reset code: " + tok,
			HTML:    "<h1>Reset code</h1><p>" + tok + "</p>",
		}
		if err := svc.Mailer.Send(ctx, msg); err != nil {
			logger.PrintError(err)
		}

		http.Redirect(w, r, svc.Path("account/forgotten_password")+"?status=email-sent", http.StatusSeeOther)
	}
}

func ForgottenPasswordPut(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(r, &input)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		email, err := tokens.FindResetPasswordTokenEmail(ctx, input.Token)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		passport, err := svc.PassportByEmail(ctx, email.String())
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		cmd := account.ResetPassword{
			Guard:            passport,
			UserID:           passport.UserID(),
			NewPassword:      input.NewPassword,
			NewPasswordCheck: input.NewPasswordCheck,
		}
		err = cmd.Validate(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "account/forgotten_password", nil) {
			return
		}

		// Only consume after manual command validation, but before execution
		// This way the token will only be consumed once we know there aren't any
		// input validation or authorisation errors
		err = tokens.ConsumeResetPasswordToken(ctx, input.Token)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = cmd.Execute(ctx, svc.Bus)
		if svc.RenderError(w, r, errors.Tracef(err), "account/forgotten_password", nil) {
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = passport.Renew()
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account/forgotten_password")+"?status=success", http.StatusSeeOther)
	}
}
