package admin

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func systemConfigRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/config", func(mux *router.ServeMux) {
		mux.Get("/", h.HTML.Handler("site/admin/system_config"), "system.config")
		mux.Post("/", systemConfigPost(h), "system.config.post")
	})
}

func systemConfigPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SystemEmail               string `form:"system-email"`
			SecurityEmail             string `form:"security-email"`
			SignUpEnabled             bool   `form:"sign-up-enabled" compare:"true"`
			SignUpAutoActivateEnabled bool   `form:"sign-up-auto-activate-enabled" compare:"true"`
			TOTPRequired              bool   `form:"totp-required" compare:"true"`
			TOTPSMSEnabled            bool   `form:"totp-sms-enabled" compare:"true"`
			GoogleSignInEnabled       bool   `form:"google-sign-in-enabled" compare:"true"`
			GoogleSignInClientID      string `form:"google-sign-in-client-id"`
			FacebookSignInEnabled     bool   `form:"facebook-sign-in-enabled" compare:"true"`
			FacebookSignInAppID       string `form:"facebook-sign-in-app-id"`
			FacebookSignInAppSecret   string `form:"facebook-sign-in-app-secret"`
			ResendAPIKey              string `form:"resend-api-key"`
			TwilioSID                 string `form:"twilio-sid"`
			TwilioToken               string `form:"twilio-token"`
			TwilioFromTel             string `form:"twilio-from-tel"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		config := h.Config(ctx)

		if !passport.System.CanUpdateConfig() {
			h.HTML.ErrorView(w, r, "can update config", app.ErrForbidden, "site/error", nil)

			return
		}

		if config.SetupRequired {
			// We force sign up and sign up auto activate enabled to be true
			// when the config is first being setup because it's required for
			// the first user to be created
			input.SignUpEnabled = true
			input.SignUpAutoActivateEnabled = true
		}

		_, err := h.Svc.System.UpdateConfig(ctx,
			passport.System,
			input.SystemEmail,
			input.SecurityEmail,
			input.SignUpEnabled,
			input.SignUpAutoActivateEnabled,
			input.TOTPRequired,
			input.TOTPSMSEnabled,
			input.GoogleSignInEnabled,
			input.GoogleSignInClientID,
			input.FacebookSignInEnabled,
			input.FacebookSignInAppID,
			input.FacebookSignInAppSecret,
			input.ResendAPIKey,
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
