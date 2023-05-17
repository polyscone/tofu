package account

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func Login(svc *handler.Services, mux *router.ServeMux) {
	mux.Post("/login/password", loginWithPasswordPost(svc))
	mux.Post("/login/totp", loginWithTOTPPost(svc))
	mux.Post("/login/recovery-code", loginWithRecoveryCodePost(svc))
}

func loginWithPasswordPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		cmd := account.AuthenticateWithPassword(input)
		res, err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		csrfToken, err := svc.RenewSession(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.Sessions.Set(ctx, sess.UserID, res.UserID)
		svc.Sessions.Set(ctx, sess.Email, input.Email)
		svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, res.HasVerifiedTOTP)
		svc.Sessions.Set(ctx, sess.TOTPUseSMS, res.TOTPUseSMS)
		svc.Sessions.Set(ctx, sess.IsAwaitingTOTP, res.HasVerifiedTOTP)
		svc.Sessions.Set(ctx, sess.IsAuthenticated, !res.HasVerifiedTOTP)

		svc.JSON(w, r, map[string]any{
			"csrfToken":      base64.RawURLEncoding.EncodeToString(csrfToken),
			"isAwaitingTOTP": res.HasVerifiedTOTP,
		})
	}
}

func loginWithTOTPPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		cmd := account.AuthenticateWithTOTP{
			UserID: svc.Sessions.GetString(ctx, sess.UserID),
			TOTP:   input.TOTP,
		}
		err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		csrfToken, err := svc.RenewSession(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.Sessions.Set(ctx, sess.IsAuthenticated, true)
		svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		svc.JSON(w, r, map[string]any{
			"csrfToken": base64.RawURLEncoding.EncodeToString(csrfToken),
		})
	}
}

func loginWithRecoveryCodePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RecoveryCode string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		cmd := account.AuthenticateWithRecoveryCode{
			UserID:       svc.Sessions.GetString(ctx, sess.UserID),
			RecoveryCode: input.RecoveryCode,
		}
		err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		csrfToken, err := svc.RenewSession(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.Sessions.Set(ctx, sess.IsAuthenticated, true)
		svc.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

		svc.JSON(w, r, map[string]any{
			"csrfToken": base64.RawURLEncoding.EncodeToString(csrfToken),
		})
	}
}
