package account

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/human"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

const lowRecoveryCodes = 2

func SignIn(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/sign-in", func(mux *router.ServeMux) {
		mux.Get("/", signInGet(h), "account.sign_in")
		mux.Post("/", signInPost(h), "account.sign_in.post")

		mux.Prefix("/totp", func(mux *router.ServeMux) {
			mux.Get("/", signInTOTPGet(h), "account.sign_in.totp")
			mux.Post("/", signInTOTPPost(h), "account.sign_in.totp.post")

			mux.Prefix("/reset", func(mux *router.ServeMux) {
				mux.Get("/", signInTOTPResetGet(h), "account.sign_in.totp.reset")
				mux.Post("/", signInTOTPResetPost(h), "account.sign_in.totp.reset.post")

				mux.Get("/email-sent", h.HandleView("account/totp/reset/email_sent"), "account.sign_in.totp.reset.email_sent")

				mux.Prefix("/request", func(mux *router.ServeMux) {
					mux.Get("/", h.HandleView("account/totp/reset/request"), "account.sign_in.totp.reset.request")
					mux.Post("/", signInTOTPResetRequestPost(h), "account.sign_in.totp.reset.request.post")

					mux.Get("/sent", h.HandleView("account/totp/reset/request_sent"), "account.sign_in.totp.reset.request.sent")
				})
			})
		})

		mux.Prefix("/recovery-code", func(mux *router.ServeMux) {
			mux.Get("/", signInRecoveryCodeGet(h), "account.sign_in.recovery_code")
			mux.Post("/", signInRecoveryCodePost(h), "account.sign_in.recovery_code.post")
		})

		mux.Post("/google", signInGooglePost(h), "account.sign_in.google.post")
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
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

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
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		err := h.Account.SignInWithTOTP(ctx, user.ID, input.TOTP)
		if err != nil {
			h.ErrorView(w, r, "sign in with TOTP", err, "account/sign_in/totp", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.ErrorView(w, r, "renew session", err, "error", nil)

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

		h.Sessions.Set(ctx, sess.IsSignedIn, true)
		h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		signInSuccessRedirect(h, w, r)
	}
}

func signInTOTPResetGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/reset/verify", nil)
	}
}

func signInTOTPResetPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := h.Logger(ctx)
		config := h.Config(ctx)
		email := h.Sessions.GetString(ctx, sess.Email)

		if email == "" || !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		background.Go(func() {
			// We can't use the request context here because it will have already
			// been cancelled after the main request handler finished
			ctx := context.Background()

			tok, err := h.Repo.Web.AddTOTPResetVerifyToken(ctx, email, 2*time.Hour)
			if err != nil {
				log.Error("TOTP reset: add verify email token", "error", err)

				return
			}

			recipients := handler.EmailRecipients{
				From: config.SystemEmail,
				To:   []string{email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := h.SendEmail(ctx, recipients, "totp_reset_verify_email", vars); err != nil {
				log.Error("TOTP reset: send email", "error", err)
			}
		})

		http.Redirect(w, r, h.Path("account.sign_in.totp.reset.email_sent"), http.StatusSeeOther)
	}
}

func signInTOTPResetRequestPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindTOTPResetVerifyTokenEmail(ctx, input.Token)
		if err != nil {
			h.ErrorView(w, r, "find TOTP reset verify token email", err, "error", nil)

			return
		}

		err = h.Account.RequestTOTPReset(ctx, email)
		if err != nil {
			h.ErrorView(w, r, "request TOTP reset", err, "account/totp/reset/request", nil)

			return
		}

		err = h.Repo.Web.ConsumeTOTPResetVerifyToken(ctx, input.Token)
		if err != nil {
			h.ErrorView(w, r, "consume TOTP reset verify token", err, "error", nil)

			return
		}

		http.Redirect(w, r, h.Path("account.sign_in.totp.reset.request.sent"), http.StatusSeeOther)
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
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)

		if !h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in"), http.StatusSeeOther)

			return
		}

		err := h.Account.SignInWithRecoveryCode(ctx, user.ID, input.RecoveryCode)
		if err != nil {
			h.ErrorView(w, r, "sign in with recovery code", err, "account/sign_in/recovery_code", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.ErrorView(w, r, "renew session", err, "error", nil)

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

		h.Sessions.Set(ctx, sess.IsSignedIn, true)
		h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		signInSuccessRedirect(h, w, r)
	}
}

func signInGooglePost(h *handler.Handler) http.HandlerFunc {
	client := http.Client{Timeout: 10 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config := h.Config(ctx)

		if config.GoogleSignInClientID == "" {
			err := errors.New("Google sign in client id has not be set")
			h.ErrorView(w, r, "check config", err, "error", nil)

			return
		}

		c, err := r.Cookie("g_csrf_token")
		if err != nil {
			h.ErrorView(w, r, "get Google CSRF cookie", err, "error", nil)

			return
		}

		csrfCookieToken := c.Value
		csrfFormToken := r.PostFormValue("g_csrf_token")
		if csrfCookieToken != csrfFormToken {
			h.ErrorView(w, r, "check CSRF", csrf.ErrInvalidToken, "error", nil)

			return
		}

		// TODO: Check cache-control
		res, err := client.Get("https://www.googleapis.com/oauth2/v1/certs")
		if err != nil {
			h.ErrorView(w, r, "fetch Google OAuth2 certs", err, "error", nil)

			return
		}
		defer res.Body.Close()

		certs := make(map[string]string)
		if err := httputil.DecodeJSON(&certs, res.Body); err != nil {
			h.ErrorView(w, r, "decode Google OAuth2 certs JSON", err, "error", nil)

			return
		}

		token := r.PostFormValue("credential")
		parts := strings.Split(token, ".")
		if want, got := 3, len(parts); want != got {
			err := fmt.Errorf("want %v parts in JWT; got %v", want, got)
			h.ErrorView(w, r, "decode JWT", err, "error", nil)

			return
		}

		var header struct {
			Alg string
			Kid string // Key ID to use from Google's public keys
			Typ string
		}
		if b, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
			h.ErrorView(w, r, "decode JWT header", err, "error", nil)

			return
		} else if json.Unmarshal(b, &header); err != nil {
			h.ErrorView(w, r, "unmarshal JWT header", err, "error", nil)

			return
		}

		if header.Typ != "JWT" {
			err := fmt.Errorf("want JWT type; got %q", header.Typ)
			h.ErrorView(w, r, "check JWT header", err, "error", nil)

			return
		}

		if header.Alg != "RS256" {
			err := fmt.Errorf("want RS256 algorithm; got %q", header.Alg)
			h.ErrorView(w, r, "check JWT header", err, "error", nil)

			return
		}

		block, _ := pem.Decode([]byte([]byte(certs[header.Kid])))
		if block == nil {
			err := errors.New("unable to decode")
			h.ErrorView(w, r, "decode certificate PEM", err, "error", nil)

			return
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			h.ErrorView(w, r, "parse Google OAuth2 cert", err, "error", nil)

			return
		}

		rsaPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			err := fmt.Errorf("could not assert cert.PublicKey as %T", rsaPublicKey)
			h.ErrorView(w, r, "extract RSA public key", err, "error", nil)

			return
		}

		payload := sha256.New()
		if _, err := payload.Write([]byte(parts[0] + "." + parts[1])); err != nil {
			h.ErrorView(w, r, "new JWT payload hash", err, "error", nil)

			return
		}
		hashed := payload.Sum(nil)

		signature, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil {
			h.ErrorView(w, r, "decode JWT signature", err, "error", nil)

			return
		}

		if err := rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed, signature); err != nil {
			h.ErrorView(w, r, "check JWT signature", err, "error", nil)

			return
		}

		var claims struct {
			Aud   string // Client ID
			Iss   string // accounts.google.com or https://accounts.google.com
			Exp   int64
			Nbf   int64
			Email string
		}
		if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
			h.ErrorView(w, r, "decode JWT claims", err, "error", nil)

			return
		} else if json.Unmarshal(b, &claims); err != nil {
			h.ErrorView(w, r, "unmarshal JWT claims", err, "error", nil)

			return
		}

		if claims.Aud != config.GoogleSignInClientID {
			err := errors.New("invalid client id")
			h.ErrorView(w, r, "check JWT claims", err, "error", nil)

			return
		}

		if claims.Iss != "accounts.google.com" && claims.Iss != "https://accounts.google.com" {
			err := fmt.Errorf("invalid issuer %q", claims.Iss)
			h.ErrorView(w, r, "check JWT claims", err, "error", nil)

			return
		}

		now := time.Now().Unix()
		if claims.Exp > 0 && claims.Exp <= now {
			err := errors.New("expired")
			h.ErrorView(w, r, "check JWT claims", err, "error", nil)

			return
		}
		if claims.Nbf > 0 && claims.Nbf > now {
			err := errors.New("used too soon")
			h.ErrorView(w, r, "check JWT claims", err, "error", nil)

			return
		}

		if err := h.Account.SignInWithGoogle(ctx, claims.Email); err != nil {
			h.ErrorView(w, r, "sign in wih Google", err, "error", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.ErrorView(w, r, "renew session", err, "error", nil)

			return
		}

		if err := signInSetSession(ctx, h, w, r, claims.Email); err != nil {
			h.ErrorView(w, r, "sign in set session", err, "error", nil)

			return
		}

		if h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
			http.Redirect(w, r, h.Path("account.sign_in.totp"), http.StatusSeeOther)

			return
		}

		signInSuccessRedirect(h, w, r)
	}
}

func signInWithPasswordErrors(err error) handler.ViewDataFunc {
	return func(data *handler.ViewData) {
		var throttle *account.SignInThrottleError
		if errors.As(err, &throttle) {
			last := human.Duration(app.SignInThrottleTTL)
			wait := human.Duration(time.Until(throttle.UnlockAt))
			if wait != "" {
				wait = " in " + wait
			}

			data.ErrorMessage = fmt.Sprintf("Too many failed sign in attempts in the last %v. Please try again%v.", last, wait)
		} else {
			data.ErrorMessage = "Either this account does not exist, or your credentials are incorrect."
		}
	}
}

func signInWithPassword(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email, password string) {
	log := h.Logger(ctx)

	attempts := h.Sessions.GetInt(ctx, sess.SignInAttempts)
	lastAttemptAt := h.Sessions.GetTime(ctx, sess.LastSignInAttemptAt)
	if time.Since(lastAttemptAt) > app.SignInThrottleTTL {
		attempts = 0
		lastAttemptAt = time.Time{}
	}

	if err := h.Account.CheckSignInThrottle(attempts, lastAttemptAt); err != nil {
		err = fmt.Errorf("check session sign in throttle: %w", err)

		h.ErrorViewFunc(w, r, "sign in with password", err, "account/sign_in/password", signInWithPasswordErrors(err))

		return
	}

	err := h.Account.SignInWithPassword(ctx, email, password)
	if err != nil {
		attempts++
		lastAttemptAt = time.Now().UTC()

		h.Sessions.Set(ctx, sess.SignInAttempts, attempts)
		h.Sessions.Set(ctx, sess.LastSignInAttemptAt, lastAttemptAt)

		h.ErrorViewFunc(w, r, "sign in with password", err, "account/sign_in/password", signInWithPasswordErrors(err))

		return
	}

	h.Sessions.Delete(ctx, sess.SignInAttempts)
	h.Sessions.Delete(ctx, sess.LastSignInAttemptAt)

	if _, err := h.RenewSession(ctx); err != nil {
		h.ErrorView(w, r, "renew session", err, "error", nil)

		return
	}

	if err := signInSetSession(ctx, h, w, r, email); err != nil {
		h.ErrorView(w, r, "sign in set session", err, "error", nil)

		return
	}

	knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(password))
	if err != nil {
		log.Error("known password breach count", "error", err)
	}
	if knownBreachCount > 0 {
		h.Sessions.Set(ctx, sess.KnownPasswordBreachCount, knownBreachCount)
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
		return fmt.Errorf("find user by email: %w", err)
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

	knownBreachCount := h.Sessions.GetInt(ctx, sess.KnownPasswordBreachCount)

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
