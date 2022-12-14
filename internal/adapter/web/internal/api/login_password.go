package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountLoginWithPasswordPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string
		Password string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.AuthenticateWithPassword(input)
	res, err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	err = csrf.RenewToken(ctx)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	err = api.sessions.Renew(ctx)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	api.sessions.Set(ctx, sesskey.UserID, res.UserID)
	api.sessions.Set(ctx, sesskey.Email, input.Email)
	api.sessions.Set(ctx, sesskey.IsAwaitingTOTP, res.IsAwaitingTOTP)

	csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

	writeJSON(w, r, map[string]any{
		"csrfToken":      csrfTokenBase64,
		"isAwaitingTOTP": res.IsAwaitingTOTP,
	})
}
