package account

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/event"
)

func RegisterSignInHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /account/sign-in", signInPost(h))
	mux.HandleFunc("POST /account/sign-in/magic-link", signInMagicLinkPost(h))
	mux.HandleFunc("POST /account/sign-in/magic-link/request", signInMagicLinkRequestPost(h))
	mux.HandleFunc("POST /account/sign-in/totp", signInTOTPPost(h))
	mux.HandleFunc("POST /account/sign-in/totp/send-sms", signInTOTPSendSMSPost(h))
	mux.HandleFunc("POST /account/sign-in/recovery-code", signInRecoveryCodePost(h))
	mux.HandleFunc("POST /account/sign-in/google", signInGooglePost(h))
	mux.HandleFunc("POST /account/sign-in/facebook", signInFacebookPost(h))
}

func signInPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if err := auth.SignInWithPassword(ctx, h.Handler, input.Email, input.Password); err != nil {
			if errors.Is(err, app.ErrNotFound) || errors.Is(err, account.ErrInvalidPassword) {
				h.JSON(w, r, http.StatusBadRequest, map[string]any{
					"error": "Either your credentials are incorrect, or you're not authorised to access this application.",
				})

				return
			}

			h.ErrorJSON(w, r, "sign in with password", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}

func signInMagicLinkRequestPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.ErrorJSON(w, r, "new email", err)

			return
		}

		h.Broker.Dispatch(event.SignInMagicLinkRequested{
			Email: input.Email,
			TTL:   10 * time.Minute,
		})

		ctx := r.Context()

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		w.WriteHeader(http.StatusOK)
	}
}

func signInMagicLinkPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if _, err := auth.SignInWithMagicLink(ctx, h.Handler, input.Token); err != nil {
			h.ErrorJSON(w, r, "sign in with magic link", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}

func signInTOTPPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if err := auth.SignInWithTOTP(ctx, h.Handler, input.TOTP); err != nil {
			h.ErrorJSON(w, r, "sign in with TOTP", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}

func signInTOTPSendSMSPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)

		h.Broker.Dispatch(event.TOTPSMSRequested{
			Email: user.Email,
			Tel:   user.TOTPTel,
		})

		w.WriteHeader(http.StatusOK)
	}
}

func signInRecoveryCodePost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecoveryCode string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if err := auth.SignInWithRecoveryCode(ctx, h.Handler, input.RecoveryCode); err != nil {
			h.ErrorJSON(w, r, "sign in with recovery code", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}

func signInGooglePost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			JWT string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if _, err := auth.SignInWithGoogle(ctx, h.Handler, input.JWT); err != nil {
			h.ErrorJSON(w, r, "sign in with Google", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}

func signInFacebookPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			UserID      string
			AccessToken string
			Email       string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if _, err := auth.SignInWithFacebook(ctx, h.Handler, input.UserID, input.AccessToken, input.Email); err != nil {
			h.ErrorJSON(w, r, "sign in with Facebook", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}