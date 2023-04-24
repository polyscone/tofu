package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (ui *UI) accountActivateGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "account_activate", nil)
}

func (ui *UI) accountActivatePost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	if input.Token == "" {
		http.Redirect(w, r, ui.route("account.activate"), http.StatusSeeOther)

		return
	}

	email, err := ui.tokens.FindActivationTokenEmail(ctx, input.Token)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	cmd := account.Activate{
		Email: email.String(),
	}
	err = cmd.Validate(ctx)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	// Only consume after manual command validation, but before execution
	// This way the token will only be consumed once we know there aren't any
	// input validation or authorisation errors
	err = ui.tokens.ConsumeActivationToken(ctx, input.Token)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	err = cmd.Execute(ctx, ui.bus)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_activate", nil) {
		return
	}

	http.Redirect(w, r, ui.route("account.activate")+"?status=success", http.StatusSeeOther)
}
