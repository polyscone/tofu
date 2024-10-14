package system

import (
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/ui"
)

func RegisterConfigHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)
		mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.System.CanViewConfig() }))

		mux.HandleFunc("GET /admin/system/config", h.HTML.HandlerFunc("site/system/config"), "system.config")
		mux.HandleFunc("POST /admin/system/config", systemConfigPost(h), "system.config.post")
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
			MagicLinkSignInEnabled    bool   `form:"magic-link-sign-in-enabled" compare:"true"`
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
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		if !passport.System.CanUpdateConfig() {
			h.HTML.ErrorView(w, r, "can update config", app.ErrForbidden, "site/error", nil)

			return
		}

		_, err := h.Svc.System.UpdateConfig(ctx,
			passport.System,
			input.SystemEmail,
			input.SecurityEmail,
			input.SignUpEnabled,
			input.SignUpAutoActivateEnabled,
			input.TOTPRequired,
			input.TOTPSMSEnabled,
			input.MagicLinkSignInEnabled,
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
			h.HTML.ErrorView(w, r, "update config", err, "site/system/config", nil)

			return
		}

		h.AddFlashf(ctx, "System configuration successfully updated.")

		var redirect string
		if h.Session.IsSignedIn(ctx) {
			redirect = h.Path("system.config")
		} else {
			redirect = h.Path("account.sign_up")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
