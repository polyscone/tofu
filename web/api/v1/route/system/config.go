package system

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/api/v1/ui"
)

func RegisterConfigHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /api/v1/system/config", configGet(h))
}

func configGet(h *ui.Handler) http.HandlerFunc {
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
