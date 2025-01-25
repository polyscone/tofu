package system

import (
	"net/http"

	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterSetupHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				config := h.Config(ctx)

				userCount, err := h.Repo.Account.CountUsers(ctx)
				if err != nil {
					h.HTML.ErrorView(w, r, "count users", err, "error", nil)

					return
				}

				if !config.SetupRequired && userCount != 0 {
					http.Redirect(w, r, mux.Path("system.config"), http.StatusSeeOther)

					return
				}

				next(w, r)
			}
		})

		mux.HandleFunc("GET /system/setup", systemSetupGet(h), "system.setup")
		mux.HandleFunc("POST /system/setup", systemSetupPost(h), "system.setup.post")
	})
}

type updateEmailsGuard struct {
	canUpdateEmails bool
}

func (g updateEmailsGuard) CanUpdateEmails() bool {
	return g.canUpdateEmails
}

func systemSetupGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "system/setup", handler.Vars{
			"SMTPEnvelopeFrom": h.SMTPEnvelopeFrom,
		})
	}
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
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)

		userCount, err := h.Repo.Account.CountUsers(ctx)
		if err != nil {
			h.HTML.ErrorView(w, r, "count users", err, "error", nil)

			return
		}

		g := updateEmailsGuard{canUpdateEmails: config.SetupRequired || userCount == 0}
		_, err = h.Svc.System.UpdateEmails(ctx, g, system.UpdateEmailsInput{
			SystemEmail:   input.SystemEmail,
			SecurityEmail: input.SecurityEmail,
		})
		if err != nil {
			h.HTML.ErrorView(w, r, "update emails", err, h.Session.LastView(ctx), nil)

			return
		}

		_, err = h.Svc.Account.SignUpInitialUser(ctx, input.Email, input.Password, input.PasswordCheck, []int{h.SuperRole.ID})
		if err != nil {
			h.HTML.ErrorView(w, r, "sign up initial user", err, h.Session.LastView(ctx), nil)

			return
		}

		err = auth.SignInWithPassword(ctx, h.Handler, input.Email, input.Password)
		if err != nil {
			h.HTML.ErrorView(w, r, "sign in with password", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.system.setup.flash.completed"))

		http.Redirect(w, r, h.Path("system.config"), http.StatusSeeOther)
	}
}
