package api

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountActivatePost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	email, err := api.tokens.FindActivationTokenEmail(ctx, input.Token)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	cmd := account.Activate{
		Email: email.String(),
	}
	err = cmd.Validate(ctx)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	// Only consume after manual command validation, but before execution
	// This way the token will only be consumed once we know there aren't any
	// input validation or authorisation errors
	err = api.tokens.ConsumeActivationToken(ctx, input.Token)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	err = cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}
}
