package account

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/port/account"
)

func Login(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/login", loginGet(svc), "account.login")
	mux.Post("/login", loginPost(svc), "account.login.post")
	mux.Post("/login/totp/send-sms", loginTOTPSendSMSPost(svc), "account.login.totp.send_sms.post")
}

func loginGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/login", nil)
	}
}

func loginPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Action       string
			Email        string
			Password     string
			TOTP         string
			RecoveryCode string
		}
		err := httputil.DecodeForm(r, &input)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		actions := map[string]struct{}{
			"verify-password":      {},
			"verify-totp":          {},
			"verify-recovery-code": {},
		}
		if _, ok := actions[input.Action]; !ok {
			svc.ErrorView(w, r, errors.Tracef("invalid action %q", input.Action), "error", nil)

			return
		}

		ctx := r.Context()

		switch input.Action {
		case "verify-password":
			loginWithPassword(ctx, svc, w, r, input.Email, input.Password)

		case "verify-totp":
			cmd := account.AuthenticateWithTOTP{
				UserID: svc.Sessions.GetString(ctx, sess.UserID),
				TOTP:   input.TOTP,
			}
			err := cmd.Execute(ctx, svc.Bus)
			if svc.ErrorView(w, r, errors.Tracef(err), "account/login", nil) {
				return
			}

			_, err = svc.RenewSession(ctx)
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			svc.Sessions.Set(ctx, sess.IsAuthenticated, true)
			svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

			loginSuccessRedirect(svc, w, r)

		case "verify-recovery-code":
			cmd := account.AuthenticateWithRecoveryCode{
				UserID:       svc.Sessions.GetString(ctx, sess.UserID),
				RecoveryCode: input.RecoveryCode,
			}
			err := cmd.Execute(ctx, svc.Bus)
			if svc.ErrorView(w, r, errors.Tracef(err), "account/login", nil) {
				return
			}

			_, err = svc.RenewSession(ctx)
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			svc.Sessions.Set(ctx, sess.IsAuthenticated, true)
			svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

			loginSuccessRedirect(svc, w, r)
		}
	}
}

func loginTOTPSendSMSPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		svc.Broker.Dispatch(handler.TOTPSMSRequested{
			Email: svc.Sessions.GetString(ctx, sess.Email),
		})

		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}

func loginWithPassword(ctx context.Context, svc *handler.Services, w http.ResponseWriter, r *http.Request, email, password string) {
	cmd := account.AuthenticateWithPassword{
		Email:    email,
		Password: password,
	}
	res, err := cmd.Execute(ctx, svc.Bus)
	if err != nil {
		svc.ErrorViewFunc(w, r, errors.Tracef(err), "account/login", func(data *handler.ViewData) {
			if errors.Is(err, repo.ErrNotFound) || errors.Is(err, account.ErrNotActivated) {
				data.ErrorMessage = "Either this account does not exist, or your credentials are incorrect."
			}
		})

		return
	}

	_, err = svc.RenewSession(ctx)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	svc.Sessions.Set(ctx, sess.UserID, res.UserID)
	svc.Sessions.Set(ctx, sess.Email, email)
	svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, res.HasVerifiedTOTP)
	svc.Sessions.Set(ctx, sess.TOTPUseSMS, res.TOTPUseSMS)
	svc.Sessions.Set(ctx, sess.IsAwaitingTOTP, res.HasVerifiedTOTP)
	svc.Sessions.Set(ctx, sess.IsAuthenticated, !res.HasVerifiedTOTP)

	knownBreachCount, err := pwned.PasswordKnownBreachCount(ctx, []byte(password))
	if err != nil {
		httputil.LogError(r, err)
	}

	if knownBreachCount > 0 {
		svc.Sessions.Set(ctx, sess.PasswordKnownBreachCount, knownBreachCount)
	}

	if svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		http.Redirect(w, r, svc.Path("account.login")+"?step=totp", http.StatusSeeOther)

		return
	}

	loginSuccessRedirect(svc, w, r)
}

func loginSuccessRedirect(svc *handler.Services, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	knownBreachCount := svc.Sessions.GetInt(ctx, sess.PasswordKnownBreachCount)

	var redirect string
	if knownBreachCount > 0 {
		redirect = svc.Path("account.change_password")
	} else if r := svc.Sessions.PopString(ctx, sess.Redirect); r != "" {
		redirect = r
	} else {
		redirect = svc.Path("account.dashboard")
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
