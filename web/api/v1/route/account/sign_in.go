package account

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/api/v1/ui"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/event"
)

func RegisterSignInHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /api/v1/account/sign-in", signInPost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/magic-link", signInMagicLinkPost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/magic-link/request", signInMagicLinkRequestPost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/totp", signInTOTPPost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/totp/send-sms", signInTOTPSendSMSPost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/recovery-code", signInRecoveryCodePost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/google", signInGooglePost(h))
	mux.HandleFunc("POST /api/v1/account/sign-in/facebook", signInFacebookPost(h))
}

func signInPost(h *ui.Handler) http.HandlerFunc {
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
					"error": h.T(ctx, i18n.M("api:account.sign_in.error")),
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

func signInMagicLinkRequestPost(h *ui.Handler) http.HandlerFunc {
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

		ctx := r.Context()

		h.Broker.Dispatch(ctx, event.SignInMagicLinkRequested{
			Email: input.Email,
			TTL:   10 * time.Minute,
		})

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		w.WriteHeader(http.StatusOK)
	}
}

func signInMagicLinkPost(h *ui.Handler) http.HandlerFunc {
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

func signInTOTPPost(h *ui.Handler) http.HandlerFunc {
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

func signInTOTPSendSMSPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)

		h.Broker.Dispatch(ctx, event.TOTPSMSRequested{
			Email: user.Email,
			Tel:   user.TOTPTel,
		})

		w.WriteHeader(http.StatusOK)
	}
}

func signInRecoveryCodePost(h *ui.Handler) http.HandlerFunc {
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

func signInGooglePost(h *ui.Handler) http.HandlerFunc {
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

func signInFacebookPost(h *ui.Handler) http.HandlerFunc {
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
