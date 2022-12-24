package api

import (
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/internal/passport"
	"github.com/polyscone/tofu/internal/adapter/web/internal/smtp"
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
	if writeError(w, r, err) {
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
		Token       string
		NewPassword string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	// TODO: Consume should return a sentinel error so we can respond with 400
	email, err := api.tokens.ConsumeResetPasswordToken(ctx, input.Token)
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
		Guard:       passport.New(guardClaims, nil, nil),
		UserID:      user.ID,
		NewPassword: input.NewPassword,
	}
	err = cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}
}
