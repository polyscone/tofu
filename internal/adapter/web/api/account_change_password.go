package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/sesskey"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountChangePasswordPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OldPassword      string
		NewPassword      string
		NewPasswordCheck string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.ChangePassword{
		Guard:            api.passport(ctx),
		UserID:           api.sessions.GetString(ctx, sesskey.UserID),
		OldPassword:      input.OldPassword,
		NewPassword:      input.NewPassword,
		NewPasswordCheck: input.NewPasswordCheck,
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

	csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

	writeJSON(w, r, map[string]any{
		"csrfToken": csrfTokenBase64,
	})
}
