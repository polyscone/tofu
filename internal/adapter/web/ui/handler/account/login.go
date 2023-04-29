package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func Login(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/login", loginGet(svc), "account.login")
	mux.Post("/login", loginPost(svc), "account.login.post")

	svc.SetViewVars("account/login", handler.Vars{
		"IsAccountUnactivated": false,
	})
}

func loginGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/login", nil)
	}
}

func loginPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		err := httputil.DecodeForm(r, &input)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		cmd := account.AuthenticateWithPassword(input)
		res, err := cmd.Execute(ctx, svc.Bus)
		switch {
		case errors.Is(err, account.ErrNotActivated):
			svc.RenderError(w, r, errors.Tracef(err), "account/login", handler.Vars{
				"IsAccountUnactivated": true,
			})

			return

		case svc.RenderError(w, r, errors.Tracef(err), "error", nil):
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = svc.Sessions.Renew(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.UserID, res.UserID)
		svc.Sessions.Set(ctx, sess.Email, cmd.Email)
		svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, res.HasVerifiedTOTP)
		svc.Sessions.Set(ctx, sess.IsAwaitingTOTP, res.HasVerifiedTOTP)
		svc.Sessions.Set(ctx, sess.IsAuthenticated, !res.HasVerifiedTOTP)

		http.Redirect(w, r, svc.Path("account.dashboard"), http.StatusSeeOther)
	}
}
