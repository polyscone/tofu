package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/csrf"
)

func (api *API) csrfGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

	writeJSON(w, r, map[string]any{
		"csrfToken": csrfTokenBase64,
	})
}
