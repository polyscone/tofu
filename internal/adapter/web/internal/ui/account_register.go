package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
)

func (app *App) accountRegisterGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app.render(w, r, http.StatusOK, "account_register", func(data *renderData) {
		data.Register.Email = app.sessions.PopString(ctx, "RegisterEmail")
	})
}

func (app *App) accountRegisterPost(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.NewV4()
	if app.renderError(w, r, errors.Tracef(err)) {
		return
	}

	ctx := r.Context()

	cmd := account.Register{
		UserID:        id.String(),
		Email:         r.PostFormValue("email"),
		Password:      r.PostFormValue("password"),
		PasswordCheck: r.PostFormValue("password"),
	}
	err = cmd.Execute(ctx, app.bus)
	switch {
	case errors.Is(err, port.ErrInvalidInput):
		app.render(w, r, http.StatusBadRequest, "account_register", func(data *renderData) {
			data.Errors = err.(errors.Trace).Fields()
		})

		return

	case app.renderError(w, r, errors.Tracef(err)):
		return
	}

	app.sessions.Set(ctx, "RegisterEmail", cmd.Email)

	http.Redirect(w, r, "/account/register?status=success", http.StatusSeeOther)
}
