package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (app *App) accountLoginGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "account_login", nil)
}

func (app *App) accountLoginPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string
		Password string
	}
	if app.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.AuthenticateWithPassword(input)
	res, err := cmd.Execute(ctx, app.bus)
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

	app.sessions.Set(ctx, sesskey.UserID, res.UserID)
	app.sessions.Set(ctx, sesskey.Email, cmd.Email)
	app.sessions.Set(ctx, sesskey.IsAwaitingTOTP, res.IsAwaitingTOTP)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
