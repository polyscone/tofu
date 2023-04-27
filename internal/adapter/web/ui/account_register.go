package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account"
)

func (ui *UI) accountRegisterGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ui.render(w, r, http.StatusOK, "account_register", func(data *renderData) {
		data.Register.Email = ui.sessions.PopString(ctx, "RegisterEmail")
	})
}

func (ui *UI) accountRegisterPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID        string
		Email         string
		Password      string
		PasswordCheck string `form:"password"` // The UI doesn't include a check field
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	id, err := uuid.NewV4()
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}
	input.UserID = id.String()

	ctx := r.Context()

	cmd := account.Register(input)
	err = cmd.Execute(ctx, ui.bus)
	if ui.renderErrorView(w, r, errors.Tracef(err), "account_register", nil) {
		return
	}

	ui.sessions.Set(ctx, "RegisterEmail", input.Email)

	http.Redirect(w, r, ui.route("account.register")+"?status=success", http.StatusSeeOther)
}
