package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/sesskey"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (ui *UI) accountLoginGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "account_login", nil)
}

func (ui *UI) accountLoginPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string
		Password string
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.AuthenticateWithPassword(input)
	res, err := cmd.Execute(ctx, ui.bus)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_login", nil) {
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

	ui.sessions.Set(ctx, sesskey.UserID, res.UserID)
	ui.sessions.Set(ctx, sesskey.Email, cmd.Email)
	ui.sessions.Set(ctx, sesskey.HasVerifiedTOTP, res.HasVerifiedTOTP)
	ui.sessions.Set(ctx, sesskey.IsAwaitingTOTP, res.HasVerifiedTOTP)
	ui.sessions.Set(ctx, sesskey.IsAuthenticated, !res.HasVerifiedTOTP)

	http.Redirect(w, r, ui.route("account"), http.StatusSeeOther)
}
