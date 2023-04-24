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

func (app *App) accountForgottenPasswordGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "account_forgotten_password", nil)
}

func (app *App) accountForgottenPasswordPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string
	}
	if app.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	email, err := text.NewEmail(input.Email)
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	ctx := r.Context()

	tok, err := app.tokens.AddResetPasswordToken(ctx, email, 2*time.Hour)
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
	if err := app.mailer.Send(ctx, msg); err != nil {
		logger.PrintError(err)
	}

	http.Redirect(w, r, "/account/forgotten-password?status=sent", http.StatusSeeOther)
}

func (app *App) accountForgottenPasswordPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token            string
		NewPassword      string `form:"new-password"`
		NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
	}
	if app.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	email, err := app.tokens.FindResetPasswordTokenEmail(ctx, input.Token)
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	findCmd := account.FindUserByEmail{
		Email: email.String(),
	}
	user, err := findCmd.Execute(ctx, app.bus)
	if app.renderError(w, r, errors.Tracef(err)) {
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
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	// Only consume after manual command validation, but before execution
	// This way the token will only be consumed once we know there aren't any
	// input validation or authorisation errors
	err = app.tokens.ConsumeResetPasswordToken(ctx, input.Token)
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	err = cmd.Execute(ctx, app.bus)
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	err = csrf.RenewToken(ctx)
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	err = app.sessions.Renew(ctx)
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	http.Redirect(w, r, "/account/forgotten-password?status=success", http.StatusSeeOther)
}
