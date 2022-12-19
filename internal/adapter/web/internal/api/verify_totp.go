package api

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountVerifyTOTPPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TOTP string
	}
	if writeError(w, r, errors.Tracef(decodeJSON(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.VerifyTOTP{
		UserID: api.sessions.GetString(ctx, sess.UserID),
		TOTP:   input.TOTP,
	}
	err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}
}
