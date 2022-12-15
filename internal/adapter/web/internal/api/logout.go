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

	encoded := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
	data := map[string]string{"csrfToken": encoded}

	writeJSON(w, r, data)
}
