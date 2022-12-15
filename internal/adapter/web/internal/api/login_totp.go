package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sess"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountLoginWithTOTPPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TOTP string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.AuthenticateWithTOTP{
		Passport: api.passport(ctx),
		TOTP:     input.TOTP,
	}
	_, err := cmd.Execute(ctx, api.bus)
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

	api.sessions.Delete(ctx, sess.IsAwaitingMFA)

	encoded := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
	data := map[string]any{"csrfToken": encoded}

	writeJSON(w, r, data)
}
