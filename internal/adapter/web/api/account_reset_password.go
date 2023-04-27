package api

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	email, err := text.NewEmail(input.Email)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	ctx := r.Context()

	tok, err := api.tokens.AddResetPasswordToken(ctx, email, 2*time.Hour)
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
	if err := api.mailer.Send(ctx, msg); err != nil {
		logger.PrintError(err)
	}
}

func (api *API) accountResetPasswordPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token            string
		NewPassword      string
		NewPasswordCheck string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	email, err := api.tokens.FindResetPasswordTokenEmail(ctx, input.Token)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	findCmd := account.FindUserByEmail{
		Email: email.String(),
	}
	user, err := findCmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
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
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	// Only consume after manual command validation, but before execution
	// This way the token will only be consumed once we know there aren't any
	// input validation or authorisation errors
	err = api.tokens.ConsumeResetPasswordToken(ctx, input.Token)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	err = cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	err = csrf.RenewToken(ctx)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	err = api.sessions.Renew(ctx)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

	writeJSON(w, r, map[string]any{
		"csrfToken": csrfTokenBase64,
	})
}
