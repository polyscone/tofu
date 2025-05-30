package account

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/internal/csrf"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

const lowRecoveryCodes = 2

func RegisterSignInHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/sign-in", signInGet(h), "account.sign_in")

	mux.HandleFunc("GET /account/sign-in/password", signInPasswordGet(h))
	mux.HandleFunc("POST /account/sign-in/password", signInPasswordPost(h), "account.sign_in.password.post")

	mux.HandleFunc("GET /account/sign-in/magic-link", signInMagicLinkGet(h), "account.sign_in.magic_link")
	mux.HandleFunc("POST /account/sign-in/magic-link", signInMagicLinkPost(h), "account.sign_in.magic_link.post")

	mux.HandleFunc("GET /account/sign-in/magic-link/request", signInMagicLinkRequestGet(h))
	mux.HandleFunc("POST /account/sign-in/magic-link/request", signInMagicLinkRequestPost(h), "account.sign_in.magic_link.request.post")

	mux.HandleFunc("GET /account/sign-in/magic-link/email-sent", signInMagicLinkEmailSentGet(h), "account.sign_in.magic_link.request.email_sent")

	mux.HandleFunc("GET /account/sign-in/totp", signInTOTPGet(h), "account.sign_in.totp")
	mux.HandleFunc("POST /account/sign-in/totp", signInTOTPPost(h), "account.sign_in.totp.post")

	mux.HandleFunc("GET /account/sign-in/totp/reset", signInTOTPResetGet(h), "account.sign_in.totp.reset")
	mux.HandleFunc("POST /account/sign-in/totp/reset", signInTOTPResetPost(h), "account.sign_in.totp.reset.post")

	mux.HandleFunc("GET /account/sign-in/totp/email-sent", signInTOTPEmailSentGet(h), "account.sign_in.totp.reset.email_sent")

	mux.HandleFunc("GET /account/sign-in/totp/request", signInTOTPResetRequestGet(h), "account.sign_in.totp.reset.request")
	mux.HandleFunc("POST /account/sign-in/totp/request", signInTOTPResetRequestPost(h), "account.sign_in.totp.reset.request.post")

	mux.HandleFunc("GET /account/sign-in/totp/request/sent", signInTOTPResetRequestSentGet(h), "account.sign_in.totp.reset.request.sent")

	mux.HandleFunc("GET /account/sign-in/recovery-code", signInRecoveryCodeGet(h), "account.sign_in.recovery_code")
	mux.HandleFunc("POST /account/sign-in/recovery-code", signInRecoveryCodePost(h), "account.sign_in.recovery_code.post")

	mux.HandleFunc("GET /account/sign-in/google", signInGoogleGet(h))
	mux.HandleFunc("POST /account/sign-in/google", signInGooglePost(h), "account.sign_in.google.post")

	mux.HandleFunc("GET /account/sign-in/facebook", signInFacebookGet(h))
	mux.HandleFunc("POST /account/sign-in/facebook", signInFacebookPost(h), "account.sign_in.facebook.post")
}

func signInGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Session.IsSignedIn(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/sign_in/web_form", nil)
	}
}

func signInPasswordGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.PathQuery(r, "account.sign_in"), http.StatusSeeOther)
	}
}

func signInPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string `form:"email"`
			Password string `form:"password"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		signInWithPassword(ctx, h, w, r, input.Email, input.Password)
	}
}

func signInMagicLinkRequestGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}
}

func signInMagicLinkRequestPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.HTML.ErrorView(w, r, "new email", err, h.Session.LastView(ctx), nil)

			return
		}

		ttl := 10 * time.Minute
		h.Broker.Dispatch(ctx, event.SignInMagicLinkRequested{
			Email: input.Email,
			TTL:   ttl,
		})

		qs := fmt.Sprintf("?ttl=%d", ttl)

		http.Redirect(w, r, h.Path("account.sign_in.magic_link.request.email_sent")+qs, http.StatusSeeOther)
	}
}

func signInMagicLinkEmailSentGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/sign_in/magic_link_sent", nil)
	}
}

func signInMagicLinkGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Session.IsSignedIn(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/sign_in/magic_link", nil)
	}
}

func signInMagicLinkPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string `form:"token"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		if input.Token == "" {
			http.Redirect(w, r, h.Path("account.sign_in.magic_link"), http.StatusSeeOther)

			return
		}

		ctx := r.Context()

		signedIn, err := auth.SignInWithMagicLink(ctx, h.Handler, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "sign in with magic link", err, h.Session.LastView(ctx), nil)

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

		if !h.Session.IsAwaitingTOTP(ctx) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if h.Session.IsSignedIn(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/sign_in/totp", nil)
	}
}

func signInTOTPPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)

		if !h.Session.IsAwaitingTOTP(ctx) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if err := auth.SignInWithTOTP(ctx, h.Handler, input.TOTP); err != nil {
			h.HTML.ErrorView(w, r, "sign in with TOTP", err, h.Session.LastView(ctx), nil)

			return
		}

		switch {
		case len(user.HashedRecoveryCodes) == 0:
			h.AddFlashImportantf(ctx, i18n.M("site.account.sign_in.flash.no_more_recovery_codes"))

		case len(user.HashedRecoveryCodes) <= lowRecoveryCodes:
			h.AddFlashImportantf(ctx, i18n.M("site.account.sign_in.flash.low_recovery_codes"))
		}

		signInSuccessRedirect(h, w, r)
	}
}

func signInTOTPResetGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Session.IsAwaitingTOTP(ctx) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/totp/reset/verify", nil)
	}
}

func signInTOTPResetPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)
		config := h.Config(ctx)
		email := h.Session.Email(ctx)

		if email == "" || !h.Session.IsAwaitingTOTP(ctx) {
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

			background.Go(func() {
				vars := handler.Vars{
					"Token":            tok,
					"RequestReviewURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.sign_in.totp.reset.request"), tok),
				}
				if err := h.SendEmail(ctx, config.SystemEmail, email, "totp_reset_verify_email", vars); err != nil {
					logger.Error("TOTP reset: send email", "error", err)
				}
			})
		})

		http.Redirect(w, r, h.Path("account.sign_in.totp.reset.email_sent"), http.StatusSeeOther)
	}
}

func signInTOTPEmailSentGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/totp/reset/email_sent", nil)
	}
}

func signInTOTPResetRequestGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/totp/reset/request", nil)
	}
}

func signInTOTPResetRequestPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string `form:"token"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindTOTPResetVerifyTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find TOTP reset verify token email", err, "error", nil)

			return
		}

		_, err = h.Svc.Account.RequestTOTPReset(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "request TOTP reset", err, h.Session.LastView(ctx), nil)

			return
		}

		err = h.Repo.Web.ConsumeTOTPResetVerifyToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume TOTP reset verify token", err, "error", nil)

			return
		}

		http.Redirect(w, r, h.Path("account.sign_in.totp.reset.request.sent"), http.StatusSeeOther)
	}
}

func signInTOTPResetRequestSentGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/totp/reset/request_sent", nil)
	}
}

func signInRecoveryCodeGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Session.IsAwaitingTOTP(ctx) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if h.Session.IsSignedIn(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/sign_in/recovery_code", nil)
	}
}

func signInRecoveryCodePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecoveryCode string `form:"recovery-code"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)

		if !h.Session.IsAwaitingTOTP(ctx) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		if err := auth.SignInWithRecoveryCode(ctx, h.Handler, input.RecoveryCode); err != nil {
			h.HTML.ErrorView(w, r, "sign in with recovery code", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashImportantf(ctx, i18n.M("site.account.sign_in.flash.totp_if_device_lost"))

		switch {
		case len(user.HashedRecoveryCodes) == 0:
			h.AddFlashImportantf(ctx, i18n.M("site.account.sign_in.flash.no_more_recovery_codes"))

		case len(user.HashedRecoveryCodes) <= lowRecoveryCodes:
			h.AddFlashImportantf(ctx, i18n.M("site.account.sign_in.flash.low_recovery_codes"))
		}

		signInSuccessRedirect(h, w, r)
	}
}

func signInGoogleGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}
}

func signInGooglePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		c, err := r.Cookie("g_csrf_token")
		if err != nil {
			h.HTML.ErrorView(w, r, "get Google CSRF cookie", err, "error", nil)

			return
		}

		csrfCookieToken := c.Value
		csrfFormToken := r.PostFormValue("g_csrf_token")
		if csrfCookieToken != csrfFormToken {
			h.HTML.ErrorView(w, r, "check CSRF", csrf.ErrInvalidToken, "error", nil)

			return
		}

		jwt := r.PostFormValue("credential")
		signedIn, err := auth.SignInWithGoogle(ctx, h.Handler, jwt)
		if err != nil {
			if errors.Is(err, account.ErrGoogleSignUpDisabled) {
				h.AddFlashErrorf(ctx, i18n.M("site:flash.bad_sign_in"))

				http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

				return
			}

			h.HTML.ErrorView(w, r, "sign in with Google", err, "error", nil)

			return
		}

		if signedIn {
			signInSuccessRedirect(h, w, r)
		} else {
			http.Redirect(w, r, h.Path("account.verify.success"), http.StatusSeeOther)
		}
	}
}

func signInFacebookGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)
	}
}

func signInFacebookPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			UserID      string `form:"user-id"`
			AccessToken string `form:"access-token"`
			Email       string `form:"email"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		signedIn, err := auth.SignInWithFacebook(ctx, h.Handler, input.UserID, input.AccessToken, input.Email)
		if err != nil {
			if errors.Is(err, account.ErrFacebookSignUpDisabled) {
				h.AddFlashErrorf(ctx, i18n.M("site:flash.bad_sign_in"))

				http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

				return
			}

			h.HTML.ErrorView(w, r, "sign in with Facebook", err, "error", nil)

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
	if err := auth.SignInWithPassword(ctx, h.Handler, email, password); err != nil {
		h.HTML.ErrorViewFunc(w, r, "sign in with password", err, h.Session.LastView(ctx), func(data *handler.ViewData) error {
			var throttle *account.SignInThrottleError
			if errors.As(err, &throttle) {
				data.ErrorMessage = h.T(ctx, i18n.M("site:account.sign_in.throttled.error",
					"in_last", throttle.InLast,
					"unlock_in", throttle.UnlockIn,
				))
			} else {
				data.ErrorMessage = h.T(ctx, i18n.M("site:account.sign_in.error"))
			}

			return nil
		})

		return
	}

	signInSuccessRedirect(h, w, r)
}

func signInSuccessRedirect(h *ui.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.Session.IsAwaitingTOTP(ctx) {
		http.Redirect(w, r, h.Path("account.sign_in.totp"), http.StatusSeeOther)

		return
	}

	if h.Session.KnownPasswordBreachCount(ctx) > 0 {
		http.Redirect(w, r, h.Path("account.change_password"), http.StatusSeeOther)

		return
	}

	config := h.Config(ctx)
	user := h.User(ctx)

	if !config.TOTPRequired && !user.HasActivatedTOTP() {
		h.AddFlashWarningf(ctx, i18n.M("site.account.sign_in.flash.consider_totp"))
	}

	if redirect := h.Session.PopRedirect(ctx); redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)

		return
	}

	http.Redirect(w, r, h.Path("account.dashboard"), http.StatusSeeOther)
}
