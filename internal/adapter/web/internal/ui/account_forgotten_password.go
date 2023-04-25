package ui

import (
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/internal/passport"
	"github.com/polyscone/tofu/internal/adapter/web/internal/smtp"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

func (ui *UI) accountForgottenPasswordGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "account_forgotten_password", nil)
}

func (ui *UI) accountForgottenPasswordPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	email, err := text.NewEmail(input.Email)
	stop := ui.renderErrorView(w, r, errors.Tracef(err), "account_forgotten_password", func(data *renderData) {
		data.Errors = errors.Map{"email": err}
	})
	if stop {
		return
	}

	ctx := r.Context()

	tok, err := ui.tokens.AddResetPasswordToken(ctx, email, 2*time.Hour)
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
	if err := ui.mailer.Send(ctx, msg); err != nil {
		logger.PrintError(err)
	}

	http.Redirect(w, r, ui.route("account.forgottenPassword")+"?status=email-sent", http.StatusSeeOther)
}

func (ui *UI) accountForgottenPasswordPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token            string
		NewPassword      string
		NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	email, err := ui.tokens.FindResetPasswordTokenEmail(ctx, input.Token)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	findCmd := account.FindUserByEmail{
		Email: email.String(),
	}
	user, err := findCmd.Execute(ctx, ui.bus)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	guardClaims := []string{user.ID}

	cmd := account.ResetPassword{
		Guard:            passport.New(guardClaims, nil, nil),
		UserID:           user.ID,
		NewPassword:      input.NewPassword,
		NewPasswordCheck: input.NewPasswordCheck,
	}
	err = cmd.Validate(ctx)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_forgotten_password", nil) {
		return
	}

	// Only consume after manual command validation, but before execution
	// This way the token will only be consumed once we know there aren't any
	// input validation or authorisation errors
	err = ui.tokens.ConsumeResetPasswordToken(ctx, input.Token)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	err = cmd.Execute(ctx, ui.bus)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_forgotten_password", nil) {
		return
	}

	err = csrf.RenewToken(ctx)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	err = ui.sessions.Renew(ctx)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	http.Redirect(w, r, ui.route("account.forgottenPassword")+"?status=success", http.StatusSeeOther)
}
