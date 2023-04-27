package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/sesskey"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountDisableTOTPPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TOTP string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.DisableTOTP{
		Guard:  api.passport(ctx),
		UserID: api.sessions.GetString(ctx, sesskey.UserID),
		TOTP:   input.TOTP,
	}
	err := cmd.Execute(ctx, api.bus)
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

	api.sessions.Delete(ctx, sesskey.IsAwaitingTOTP)

	csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

	writeJSON(w, r, map[string]any{
		"csrfToken": csrfTokenBase64,
	})
}
