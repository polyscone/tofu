package api

import (
	"encoding/base32"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountSetupTOTPPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	passport := api.passport(ctx)

	cmd := account.SetupTOTP{
		Guard:  passport,
		UserID: passport.UserID(),
	}
	res, err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(res.Key)

	writeJSON(w, r, map[string]any{
		"key": keyBase32,
	})
}
