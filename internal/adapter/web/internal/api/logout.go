package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func (api *API) accountLogoutPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := csrf.RenewToken(ctx)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	api.sessions.Destroy(r.Context())

	csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

	writeJSON(w, r, map[string]any{
		"csrfToken": csrfTokenBase64,
	})
}
