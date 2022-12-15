package api

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/csrf"
)

func (api *API) csrfGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	encoded := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
	data := map[string]string{"csrfToken": encoded}

	writeJSON(w, r, data)
}
