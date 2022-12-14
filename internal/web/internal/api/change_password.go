package api

import (
	"net/http"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func (api *API) accountChangePasswordPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		NewPassword string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()
	passport := api.passport(ctx)

	cmd := account.ChangePassword{
		Guard:       passport,
		UserID:      passport.UserID(),
		NewPassword: input.NewPassword,
	}
	err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}
}
