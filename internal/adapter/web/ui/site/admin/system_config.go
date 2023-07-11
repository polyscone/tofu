package admin

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func SystemConfig(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/config", func(mux *router.ServeMux) {
		mux.Get("/", h.HTML.Handler("site/admin/system_config"), "system.config")
		mux.Post("/", systemConfigPost(h), "system.config.post")
	})
}

func systemConfigPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SystemEmail          string
			SecurityEmail        string
			TOTPRequired         bool `compare:"true"`
			GoogleSignInEnabled  bool `compare:"true"`
			GoogleSignInClientID string
			TwilioSID            string
			TwilioToken          string
			TwilioFromTel        string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		if !passport.System.CanUpdateConfig() {
			h.HTML.ErrorView(w, r, "can update config", app.ErrUnauthorised, "site/error", nil)

			return
		}

		_, err := h.Svc.System.UpdateConfig(ctx,
			passport.System,
			input.SystemEmail,
			input.SecurityEmail,
			input.TOTPRequired,
			input.GoogleSignInEnabled,
			input.GoogleSignInClientID,
			input.TwilioSID,
			input.TwilioToken,
			input.TwilioFromTel,
		)
		if err != nil {
			h.HTML.ErrorView(w, r, "update config", err, "site/admin/system_config", nil)

			return
		}

		h.AddFlashf(ctx, "System configuration successfully updated.")

		var redirect string
		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			redirect = h.Path("system.config")
		} else {
			redirect = h.Path("account.sign_up")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
