package api

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountRegisterPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID        string
		Email         string
		Password      string
		PasswordCheck string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.Register(input)
	err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}
}
