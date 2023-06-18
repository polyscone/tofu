package admin

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func SystemConfig(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/config", func(mux *router.ServeMux) {
		mux.Get("/", systemConfigGet(h), "system.config")
		mux.Post("/", systemConfigPost(h), "system.config.post")
	})
}

func systemConfigGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "admin/system_config", nil)
	}
}

func systemConfigPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SystemEmail          string
			GoogleSignInClientID string
			TwilioSID            string
			TwilioToken          string
			TwilioFromTel        string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		if !passport.CanUpdateConfig() {
			h.ErrorView(w, r, "can update config", app.ErrUnauthorised, "error", nil)

			return
		}

		_, err := h.System.UpdateConfig(ctx, passport,
			input.SystemEmail,
			input.GoogleSignInClientID,
			input.TwilioSID,
			input.TwilioToken,
			input.TwilioFromTel,
		)
		if err != nil {
			h.ErrorView(w, r, "update config", err, "admin/system_config", nil)

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
