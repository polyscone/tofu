package api

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountRegenerateRecoveryCodesPut(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cmd := account.RegenerateRecoveryCodes{
		Guard:  api.passport(ctx),
		UserID: api.sessions.GetString(ctx, sesskey.UserID),
	}
	res, err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	writeJSON(w, r, map[string]any{
		"recoveryCodes": res.RecoveryCodes,
	})
}
