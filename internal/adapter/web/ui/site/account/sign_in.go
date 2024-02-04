package account

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/auth"
	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/human"
)

const lowRecoveryCodes = 2

func signInRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/sign-in", func(mux *router.ServeMux) {
		mux.Get("/", signInGet(h), "account.sign_in")

		mux.Post("/password", signInPasswordPost(h), "account.sign_in.password.post")

		mux.Prefix("/magic-link", func(mux *router.ServeMux) {
			mux.Get("/", signInMagicLinkGet(h), "account.sign_in.magic_link")
			mux.Post("/", signInMagicLinkPost(h), "account.sign_in.magic_link.post")
			mux.Post("/request", signInMagicLinkRequestPost(h), "account.sign_in.magic_link.request.post")
			mux.Get("/email-sent", h.HTML.Handler("site/account/sign_in/magic_link_sent"), "account.sign_in.magic_link.request.email_sent")
		})

		mux.Prefix("/totp", func(mux *router.ServeMux) {
			mux.Get("/", signInTOTPGet(h), "account.sign_in.totp")
			mux.Post("/", signInTOTPPost(h), "account.sign_in.totp.post")

			mux.Prefix("/reset", func(mux *router.ServeMux) {
				mux.Get("/", signInTOTPResetGet(h), "account.sign_in.totp.reset")
				mux.Post("/", signInTOTPResetPost(h), "account.sign_in.totp.reset.post")

				mux.Get("/email-sent", h.HTML.Handler("site/account/totp/reset/email_sent"), "account.sign_in.totp.reset.email_sent")

				mux.Prefix("/request", func(mux *router.ServeMux) {
					mux.Get("/", h.HTML.Handler("site/account/totp/reset/request"), "account.sign_in.totp.reset.request")
					mux.Post("/", signInTOTPResetRequestPost(h), "account.sign_in.totp.reset.request.post")

					mux.Get("/sent", h.HTML.Handler("site/account/totp/reset/request_sent"), "account.sign_in.totp.reset.request.sent")
				})
			})
		})

		mux.Prefix("/recovery-code", func(mux *router.ServeMux) {
			mux.Get("/", signInRecoveryCodeGet(h), "account.sign_in.recovery_code")
			mux.Post("/", signInRecoveryCodePost(h), "account.sign_in.recovery_code.post")
		})

		mux.Post("/google", signInGooglePost(h), "account.sign_in.google.post")
		mux.Post("/facebook", signInFacebookPost(h), "account.sign_in.facebook.post")
	})
}

func signInGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.HTML.View(w, r, http.StatusOK, "site/account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/sign_in/web_form", nil)
	}
}

func signInPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string `form:"email"`
			Password string `form:"password"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		signInWithPassword(ctx, h, w, r, input.Email, input.Password)
	}
}

func signInMagicLinkRequestPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.HTML.ErrorView(w, r, "new email", err, "site/account/sign_in/web_form", nil)

			return
		}

		ttl := 10 * time.Minute
		h.Broker.Dispatch(event.SignInMagicLinkRequested{
			Email: input.Email,
			TTL:   ttl,
		})

		qs := "?ttl=" + human.Duration(ttl)

		http.Redirect(w, r, h.Path("account.sign_in.magic_link.request.email_sent")+qs, http.StatusSeeOther)
	}
}

func signInMagicLinkGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.HTML.View(w, r, http.StatusOK, "site/account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/sign_in/magic_link", nil)
	}
}

func signInMagicLinkPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string `form:"token"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.sign_in.magic_link"), http.StatusSeeOther)

			return
		}

		ctx := r.Context()

		signedIn, err := auth.SignInWithMagicLink(ctx, h.Handler, w, r, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "sign in with magic link", err, "site/account/sign_in/magic_link", nil)

			return
		}

		if signedIn {
			signInSuccessRedirect(h, w, r)
		} else {
			http.Redirect(w, r, h.Path("account.verify.success"), http.StatusSeeOther)
		}
	}
}

func signInTOTPGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.HTML.View(w, r, http.StatusOK, "site/account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/sign_in/totp", nil)
	}
}

func signInTOTPPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if err := auth.SignInWithTOTP(ctx, h.Handler, w, r, input.TOTP); err != nil {
			h.HTML.ErrorView(w, r, "sign in with TOTP", err, "site/account/sign_in/totp", nil)

			return
		}

		switch {
		case len(user.HashedRecoveryCodes) == 0:
			h.AddFlashImportantf(ctx, `
				You've run out of recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)

		case len(user.HashedRecoveryCodes) <= lowRecoveryCodes:
			h.AddFlashImportantf(ctx, `
				You're running low on recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)
		}

		signInSuccessRedirect(h, w, r)
	}
}

func signInTOTPResetGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/reset/verify", nil)
	}
}

func signInTOTPResetPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)
		config := h.Config(ctx)
		email := h.Sessions.GetString(ctx, sess.Email)

		if email == "" || !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		background.Go(func() {
			ctx := context.Background()

			tok, err := h.Repo.Web.AddTOTPResetVerifyToken(ctx, email, 2*time.Hour)
			if err != nil {
				logger.Error("TOTP reset: add verify email token", "error", err)

				return
			}

			vars := handler.Vars{"Token": tok}
			if err := h.SendEmail(ctx, config.SystemEmail, email, "site/totp_reset_verify_email", vars); err != nil {
				logger.Error("TOTP reset: send email", "error", err)
			}
		})

		http.Redirect(w, r, h.Path("account.sign_in.totp.reset.email_sent"), http.StatusSeeOther)
	}
}

func signInTOTPResetRequestPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string `form:"token"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindTOTPResetVerifyTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find TOTP reset verify token email", err, "site/error", nil)

			return
		}

		err = h.Svc.Account.RequestTOTPReset(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "request TOTP reset", err, "site/account/totp/reset/request", nil)

			return
		}

		err = h.Repo.Web.ConsumeTOTPResetVerifyToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume TOTP reset verify token", err, "site/error", nil)

			return
		}

		http.Redirect(w, r, h.Path("account.sign_in.totp.reset.request.sent"), http.StatusSeeOther)
	}
}

func signInRecoveryCodeGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.HTML.View(w, r, http.StatusOK, "site/account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/sign_in/recovery_code", nil)
	}
}

func signInRecoveryCodePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecoveryCode string `form:"recovery-code"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if err := auth.SignInWithRecoveryCode(ctx, h.Handler, w, r, input.RecoveryCode); err != nil {
			h.HTML.ErrorView(w, r, "sign in with recovery code", err, "site/account/sign_in/recovery_code", nil)

			return
		}

		h.AddFlashImportantf(ctx, `
			If you've lost your authentication device
			<a href="`+h.Path("account.totp.disable")+`">disable two-factor authentication</a>
			to avoid getting locked out of your account.
		`)

		switch {
		case len(user.HashedRecoveryCodes) == 0:
			h.AddFlashImportantf(ctx, `
				You've run out of recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)

		case len(user.HashedRecoveryCodes) <= lowRecoveryCodes:
			h.AddFlashImportantf(ctx, `
				You're running low on recovery codes.<br>
				We recommend
				<a href="`+h.Path("account.totp.recovery_codes")+`">generating new ones</a>
				as soon as you can.
			`)
		}

		signInSuccessRedirect(h, w, r)
	}
}

func signInGooglePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		c, err := r.Cookie("g_csrf_token")
		if err != nil {
			h.HTML.ErrorView(w, r, "get Google CSRF cookie", err, "site/error", nil)

			return
		}

		csrfCookieToken := c.Value
		csrfFormToken := r.PostFormValue("g_csrf_token")
		if csrfCookieToken != csrfFormToken {
			h.HTML.ErrorView(w, r, "check CSRF", csrf.ErrInvalidToken, "site/error", nil)

			return
		}

		jwt := r.PostFormValue("credential")
		signedIn, err := auth.SignInWithGoogle(ctx, h.Handler, w, r, jwt)
		if err != nil {
			if errors.Is(err, account.ErrGoogleSignUpDisabled) {
				h.AddFlashErrorf(ctx, "Either your credentials are incorrect, or you're not authorised to access this application.")

				http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

				return
			}

			h.HTML.ErrorView(w, r, "sign in with Google", err, "site/error", nil)

			return
		}

		if signedIn {
			signInSuccessRedirect(h, w, r)
		} else {
			http.Redirect(w, r, h.Path("account.verify.success"), http.StatusSeeOther)
		}
	}
}

func signInFacebookPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			UserID      string `form:"user-id"`
			AccessToken string `form:"access-token"`
			Email       string `form:"email"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		signedIn, err := auth.SignInWithFacebook(ctx, h.Handler, w, r, input.UserID, input.AccessToken, input.Email)
		if err != nil {
			if errors.Is(err, account.ErrFacebookSignUpDisabled) {
				h.AddFlashErrorf(ctx, "Either your credentials are incorrect, or you're not authorised to access this application.")

				http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

				return
			}

			h.HTML.ErrorView(w, r, "sign in with Facebook", err, "site/error", nil)

			return
		}

		if signedIn {
			signInSuccessRedirect(h, w, r)
		} else {
			http.Redirect(w, r, h.Path("account.verify.success"), http.StatusSeeOther)
		}
	}
}

func signInWithPassword(ctx context.Context, h *ui.Handler, w http.ResponseWriter, r *http.Request, email, password string) {
	if err := auth.SignInWithPassword(ctx, h.Handler, w, r, email, password); err != nil {
		h.HTML.ErrorViewFunc(w, r, "sign in with password", err, "site/account/sign_in/web_form", func(data *handler.ViewData) {
			var throttle *account.SignInThrottleError
			if errors.As(err, &throttle) {
				wait := human.Duration(throttle.UnlockIn)
				if wait != "" {
					wait = " in " + wait
				}

				data.ErrorMessage = fmt.Sprintf("Too many failed sign in attempts in the last %v. Please try again%v.", human.Duration(throttle.InLast), wait)
			} else {
				data.ErrorMessage = "Either your credentials are incorrect, or you're not authorised to access this application."
			}
		})

		return
	}

	signInSuccessRedirect(h, w, r)
}

func signInSuccessRedirect(h *ui.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		http.Redirect(w, r, h.Path("account.sign_in.totp"), http.StatusSeeOther)

		return
	}

	if h.Sessions.GetInt(ctx, sess.KnownPasswordBreachCount) > 0 {
		http.Redirect(w, r, h.Path("account.change_password"), http.StatusSeeOther)

		return
	}

	if redirect := h.Sessions.PopString(ctx, sess.Redirect); redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)

		return
	}

	http.Redirect(w, r, h.Path("account.dashboard"), http.StatusSeeOther)
}
