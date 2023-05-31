package account

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
	"github.com/polyscone/tofu/internal/repo"
)

const lowRecoveryCodes = 2

func SignIn(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/sign-in", func(mux *router.ServeMux) {
		mux.Get("/", signInGet(svc), "account.sign_in")
		mux.Post("/", signInPost(svc), "account.sign_in.post")

		mux.Prefix("/totp", func(mux *router.ServeMux) {
			mux.Get("/", signInTOTPGet(svc), "account.sign_in.totp")
			mux.Post("/", signInTOTPPost(svc), "account.sign_in.totp.post")
		})

		mux.Prefix("/recovery-code", func(mux *router.ServeMux) {
			mux.Get("/", signInRecoveryCodeGet(svc), "account.sign_in.recovery_code")
			mux.Post("/", signInRecoveryCodePost(svc), "account.sign_in.recovery_code.post")
		})
	})
}

func signInGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.IsSignedIn) {
			svc.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/sign_in/password", nil)
	}
}

func signInPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		signInWithPassword(ctx, svc, w, r, input.Email, input.Password)
	}
}

func signInTOTPGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.IsSignedIn) {
			svc.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/sign_in/totp", nil)
	}
}

func signInTOTPPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		err = svc.Account.SignInWithTOTP(ctx, userID, input.TOTP)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/sign_in/totp", nil) {
			return
		}

		_, err = svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		if len(user.RecoveryCodes) <= lowRecoveryCodes {
			svc.FlashImportant(ctx, `
				You are running low on recovery codes.<br>
				We recommend
				<a href="`+svc.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)
		}

		svc.Sessions.Set(ctx, sess.IsSignedIn, true)
		svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		signInSuccessRedirect(svc, w, r)
	}
}

func signInRecoveryCodeGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.IsSignedIn) {
			svc.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/sign_in/recovery_code", nil)
	}
}

func signInRecoveryCodePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecoveryCode string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		err = svc.Account.SignInWithRecoveryCode(ctx, userID, input.RecoveryCode)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/sign_in/recovery_code", nil) {
			return
		}

		_, err = svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		flash := `
			If you have lost your authentication device
			<a href="` + svc.Path("account.totp.disable") + `">disable two-factor authentication</a>
			to avoid getting locked out of your account.
		`

		if len(user.RecoveryCodes) <= lowRecoveryCodes {
			flash += `
				<br>
				<br>
				You are also running low on recovery codes.<br>
				If you still have your authentication device you can
				<a href="` + svc.Path("account.totp.recovery_codes") + `">generate new recovery codes</a>.
			`
		}

		svc.FlashImportant(ctx, flash)

		svc.Sessions.Set(ctx, sess.IsSignedIn, true)
		svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		signInSuccessRedirect(svc, w, r)
	}
}

func signInWithPassword(ctx context.Context, svc *handler.Services, w http.ResponseWriter, r *http.Request, email, password string) {
	err := svc.Account.SignInWithPassword(ctx, email, password)
	if err != nil {
		svc.ErrorViewFunc(w, r, errors.Tracef(err), "account/sign_in/password", func(data *handler.ViewData) {
			if errors.Is(err, app.ErrBadRequest) || errors.Is(err, repo.ErrNotFound) || errors.Is(err, account.ErrNotActivated) {
				data.ErrorMessage = "Either this account does not exist, or your credentials are incorrect."
			}
		})

		return
	}

	_, err = svc.RenewSession(ctx)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	err = signInSetSession(ctx, svc, w, r, email)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	knownBreachCount, err := pwned.PasswordKnownBreachCount(ctx, []byte(password))
	if err != nil {
		httputil.LogError(r, err)
	}

	if knownBreachCount > 0 {
		svc.Sessions.Set(ctx, sess.PasswordKnownBreachCount, knownBreachCount)
	}

	if svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		http.Redirect(w, r, svc.Path("account.sign_in.totp"), http.StatusSeeOther)

		return
	}

	signInSuccessRedirect(svc, w, r)
}

func signInSetSession(ctx context.Context, svc *handler.Services, w http.ResponseWriter, r *http.Request, email string) error {
	user, err := svc.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return errors.Tracef(err)
	}

	svc.Sessions.Set(ctx, sess.UserID, user.ID)
	svc.Sessions.Set(ctx, sess.Email, email)
	svc.Sessions.Set(ctx, sess.TOTPMethod, user.TOTPMethod)
	svc.Sessions.Set(ctx, sess.HasActivatedTOTP, user.HasActivatedTOTP())
	svc.Sessions.Set(ctx, sess.IsAwaitingTOTP, user.HasActivatedTOTP())
	svc.Sessions.Set(ctx, sess.IsSignedIn, !user.HasActivatedTOTP())

	return nil
}

func signInSuccessRedirect(svc *handler.Services, w http.ResponseWriter, r *http.Request) {
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
