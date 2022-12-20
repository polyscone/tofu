package api

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountActivatePost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token    string
		Email    string
		Password string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	// TODO: Consume should return a sentinel error so we can respond with 400
	_, err := api.tokens.ConsumeActivationToken(ctx, input.Token)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	cmd := account.Activate{
		Email:    input.Email,
		Password: input.Password,
	}
	err = cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}
}
