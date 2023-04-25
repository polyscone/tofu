package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (ui *UI) accountChangePasswordGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "account_change_password", nil)
}

func (ui *UI) accountChangePasswordPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OldPassword      string
		NewPassword      string
		NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.ChangePassword{
		Guard:            ui.passport(ctx),
		UserID:           ui.sessions.GetString(ctx, sesskey.UserID),
		OldPassword:      input.OldPassword,
		NewPassword:      input.NewPassword,
		NewPasswordCheck: input.NewPasswordCheck,
	}
	err := cmd.Execute(ctx, ui.bus)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_change_password", nil) {
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

	http.Redirect(w, r, ui.route("account.changePassword")+"?status=success", http.StatusSeeOther)
}
