package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (app *App) accountActivateGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "account_activate", nil)
}

func (app *App) accountActivatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.PostFormValue("token")
	if token == "" {
		http.Redirect(w, r, "/account/activate", http.StatusSeeOther)

		return
	}

	email, err := app.tokens.FindActivationTokenEmail(ctx, token)
	if app.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	cmd := account.Activate{
		Email: email.String(),
	}
	err = cmd.Validate(ctx)
	if app.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	// Only consume after manual command validation, but before execution
	// This way the token will only be consumed once we know there aren't any
	// input validation or authorisation errors
	err = app.tokens.ConsumeActivationToken(ctx, token)
	if app.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	err = cmd.Execute(ctx, app.bus)
	if app.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	http.Redirect(w, r, "/account/activate?status=success", http.StatusSeeOther)
}
