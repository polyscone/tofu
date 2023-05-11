package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func Login(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/login", loginGet(svc), "account.login")
	mux.Post("/login", loginPost(svc), "account.login.post")
	mux.Post("/login/totp", loginTOTPPost(svc), "account.login.totp.post")

	svc.SetViewVars("account/login", handler.Vars{
		"IsAccountUnactivated": false,
	})
}

func loginGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/login", nil)
	}
}

func loginPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		err := httputil.DecodeForm(r, &input)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		cmd := account.AuthenticateWithPassword(input)
		res, err := cmd.Execute(ctx, svc.Bus)
		if err != nil {
			svc.ErrorView(w, r, errors.Tracef(err), "account/login", handler.Vars{
				"IsAccountUnactivated": errors.Is(err, account.ErrNotActivated),
			})

			return
		}

		_, err = svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.UserID, res.UserID)
		svc.Sessions.Set(ctx, sess.Email, input.Email)
		svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, res.HasVerifiedTOTP)
		svc.Sessions.Set(ctx, sess.IsAwaitingTOTP, res.HasVerifiedTOTP)
		svc.Sessions.Set(ctx, sess.IsAuthenticated, !res.HasVerifiedTOTP)

		if svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, svc.Path("account.login")+"?step=totp", http.StatusSeeOther)

			return
		}

		loginSuccessRedirect(w, r, svc)
	}
}

func loginTOTPPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(r, &input)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		cmd := account.AuthenticateWithTOTP{
			UserID: svc.Sessions.GetString(ctx, sess.UserID),
			TOTP:   input.TOTP,
		}
		err = cmd.Execute(ctx, svc.Bus)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/login", nil) {
			return
		}

		_, err = svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.IsAuthenticated, true)
		svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		loginSuccessRedirect(w, r, svc)
	}
}

func loginSuccessRedirect(w http.ResponseWriter, r *http.Request, svc *handler.Services) {
	ctx := r.Context()

	var redirect string
	if r := svc.Sessions.PopString(ctx, sess.Redirect); r != "" {
		redirect = r
	} else {
		redirect = svc.Path("account.dashboard")
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
