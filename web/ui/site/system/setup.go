package system

import (
	"net/http"

	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/ui"
)

func RegisterSetupHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				config := h.Config(ctx)

				userCount, err := h.Repo.Account.CountUsers(ctx)
				if err != nil {
					h.HTML.ErrorView(w, r, "count users", err, "site/error", nil)

					return
				}

				if !config.SetupRequired && userCount != 0 {
					http.Redirect(w, r, mux.Path("system.config"), http.StatusSeeOther)

					return
				}

				next(w, r)
			}
		})

		mux.HandleFunc("GET /system/setup", h.HTML.HandlerFunc("site/system/setup"), "system.setup")
		mux.HandleFunc("POST /system/setup", systemSetupPost(h), "system.setup.post")
	})
}

type updateEmailsGuard struct {
	canUpdateEmails bool
}

func (g updateEmailsGuard) CanUpdateEmails() bool {
	return g.canUpdateEmails
}

func systemSetupPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email         string `form:"email"`
			Password      string `form:"password"`
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
			SystemEmail   string `form:"system-email"`
			SecurityEmail string `form:"security-email"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)

		userCount, err := h.Repo.Account.CountUsers(ctx)
		if err != nil {
			h.HTML.ErrorView(w, r, "count users", err, "site/error", nil)

			return
		}

		g := updateEmailsGuard{canUpdateEmails: config.SetupRequired || userCount == 0}
		err = h.Svc.System.UpdateEmails(ctx, g, input.SystemEmail, input.SecurityEmail)
		if err != nil {
			h.HTML.ErrorView(w, r, "update emails", err, "site/system/setup", nil)

			return
		}

		err = h.Svc.Account.SignUpInitialUser(ctx, input.Email, input.Password, input.PasswordCheck, []string{h.SuperRole.ID})
		if err != nil {
			h.HTML.ErrorView(w, r, "sign up initial user", err, "site/system/setup", nil)

			return
		}

		err = auth.SignInWithPassword(ctx, h.Handler, input.Email, input.Password)
		if err != nil {
			h.HTML.ErrorView(w, r, "sign in with password", err, "site/system/setup", nil)

			return
		}

		h.AddFlashf(ctx, "Setup completed successfully.")

		http.Redirect(w, r, h.Path("system.config"), http.StatusSeeOther)
	}
}
