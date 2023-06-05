package account

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
	"github.com/polyscone/tofu/internal/repo"
)

const lowRecoveryCodes = 2

func SignIn(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Prefix("/sign-in", func(mux *router.ServeMux) {
		mux.Get("/", signInGet(h), "account.sign_in")
		mux.Post("/", signInPost(h), "account.sign_in.post")

		mux.Prefix("/totp", func(mux *router.ServeMux) {
			mux.Get("/", signInTOTPGet(h), "account.sign_in.totp")
			mux.Post("/", signInTOTPPost(h), "account.sign_in.totp.post")
		})

		mux.Prefix("/recovery-code", func(mux *router.ServeMux) {
			mux.Get("/", signInRecoveryCodeGet(h), "account.sign_in.recovery_code")
			mux.Post("/", signInRecoveryCodePost(h), "account.sign_in.recovery_code.post")
		})
	})
}

func signInGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/sign_in/password", nil)
	}
}

func signInPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		signInWithPassword(ctx, h, w, r, input.Email, input.Password)
	}
}

func signInTOTPGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/sign_in/totp", nil)
	}
}

func signInTOTPPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		err = h.Account.SignInWithTOTP(ctx, userID, input.TOTP)
		if h.ErrorView(w, r, errors.Tracef(err), "account/sign_in/totp", nil) {
			return
		}

		_, err = h.RenewSession(ctx)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		switch {
		case len(user.RecoveryCodes) == 0:
			h.AddFlashImportantf(ctx, `
				You have run out of recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)

		case len(user.RecoveryCodes) <= lowRecoveryCodes:
			h.AddFlashImportantf(ctx, `
				You are running low on recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)
		}

		h.Sessions.Set(ctx, sess.IsSignedIn, true)
		h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		signInSuccessRedirect(h, w, r)
	}
}

func signInRecoveryCodeGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/sign_in/recovery_code", nil)
	}
}

func signInRecoveryCodePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecoveryCode string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		err = h.Account.SignInWithRecoveryCode(ctx, userID, input.RecoveryCode)
		if h.ErrorView(w, r, errors.Tracef(err), "account/sign_in/recovery_code", nil) {
			return
		}

		_, err = h.RenewSession(ctx)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.AddFlashImportantf(ctx, `
			If you have lost your authentication device
			<a href="`+h.Path("account.totp.disable")+`">disable two-factor authentication</a>
			to avoid getting locked out of your account.
		`)

		switch {
		case len(user.RecoveryCodes) == 0:
			h.AddFlashImportantf(ctx, `
				You have run out of recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)

		case len(user.RecoveryCodes) <= lowRecoveryCodes:
			h.AddFlashImportantf(ctx, `
				You are running low on recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)
		}

		h.Sessions.Set(ctx, sess.IsSignedIn, true)
		h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		signInSuccessRedirect(h, w, r)
	}
}

func signInWithPassword(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email, password string) {
	err := h.Account.SignInWithPassword(ctx, email, password)
	if err != nil {
		h.ErrorViewFunc(w, r, errors.Tracef(err), "account/sign_in/password", func(data *handler.ViewData) {
			if errors.Is(err, app.ErrBadRequest) || errors.Is(err, repo.ErrNotFound) || errors.Is(err, account.ErrNotActivated) {
				data.ErrorMessage = "Either this account does not exist, or your credentials are incorrect."
			}
		})

		return
	}

	_, err = h.RenewSession(ctx)
	if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	err = signInSetSession(ctx, h, w, r, email)
	if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	knownBreachCount, err := pwned.PasswordKnownBreachCount(ctx, []byte(password))
	if err != nil {
		httputil.LogError(r, err)
	}

	if knownBreachCount > 0 {
		h.Sessions.Set(ctx, sess.PasswordKnownBreachCount, knownBreachCount)
	}

	if h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		http.Redirect(w, r, h.Path("account.sign_in.totp"), http.StatusSeeOther)

		return
	}

	signInSuccessRedirect(h, w, r)
}

func signInSetSession(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email string) error {
	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return errors.Tracef(err)
	}

	h.Sessions.Set(ctx, sess.UserID, user.ID)
	h.Sessions.Set(ctx, sess.Email, email)
	h.Sessions.Set(ctx, sess.TOTPMethod, user.TOTPMethod)
	h.Sessions.Set(ctx, sess.HasActivatedTOTP, user.HasActivatedTOTP())
	h.Sessions.Set(ctx, sess.IsAwaitingTOTP, user.HasActivatedTOTP())
	h.Sessions.Set(ctx, sess.IsSignedIn, !user.HasActivatedTOTP())

	return nil
}

func signInSuccessRedirect(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	knownBreachCount := h.Sessions.GetInt(ctx, sess.PasswordKnownBreachCount)

	var redirect string
	if knownBreachCount > 0 {
		redirect = h.Path("account.change_password")
	} else if r := h.Sessions.PopString(ctx, sess.Redirect); r != "" {
		redirect = r
	} else {
		redirect = h.Path("account.dashboard")
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
