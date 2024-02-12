package system

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func configRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /system/config", configGet(h))
}

func configGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config := h.Config(ctx)

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

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
