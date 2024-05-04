package system

import (
	"net/http"

	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/api"
)

func RegisterConfigHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /system/config", configGet(h))
}

func configGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config := h.Config(ctx)

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, map[string]any{
			"signUpEnabled":          config.SignUpEnabled,
			"magicLinkSignInEnabled": config.MagicLinkSignInEnabled,
			"googleSignInEnabled":    config.GoogleSignInEnabled,
			"googleSignInClientId":   config.GoogleSignInClientID,
			"facebookSignInEnabled":  config.FacebookSignInEnabled,
			"facebookSignInAppId":    config.FacebookSignInAppID,
		})
	}
}
