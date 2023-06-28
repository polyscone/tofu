package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

func SignInWithPassword(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email, password string) error {
	logger := h.Logger(ctx)

	attempts := h.Sessions.GetInt(ctx, sess.SignInAttempts)
	lastAttemptAt := h.Sessions.GetTime(ctx, sess.LastSignInAttemptAt)
	if time.Since(lastAttemptAt) > app.SignInThrottleTTL {
		attempts = 0
		lastAttemptAt = time.Time{}
	}

	if err := h.Svc.Account.CheckSignInThrottle(attempts, lastAttemptAt); err != nil {
		return fmt.Errorf("check session sign in throttle: %w", err)
	}

	err := h.Svc.Account.SignInWithPassword(ctx, email, password)
	if err != nil {
		attempts++
		lastAttemptAt = time.Now().UTC()

		h.Sessions.Set(ctx, sess.SignInAttempts, attempts)
		h.Sessions.Set(ctx, sess.LastSignInAttemptAt, lastAttemptAt)

		return err
	}

	h.Sessions.Delete(ctx, sess.SignInAttempts)
	h.Sessions.Delete(ctx, sess.LastSignInAttemptAt)

	if _, err := h.RenewSession(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}

	if err := SignInSetSession(ctx, h, w, r, email); err != nil {
		return fmt.Errorf("sign in set session: %w", err)
	}

	knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(password))
	if err != nil {
		logger.Error("known password breach count", "error", err)
	}
	if knownBreachCount > 0 {
		h.Sessions.Set(ctx, sess.KnownPasswordBreachCount, knownBreachCount)
	}

	return nil
}

func SignInWithTOTP(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, totp string) error {
	user := h.User(ctx)

	err := h.Svc.Account.SignInWithTOTP(ctx, user.ID, totp)
	if err != nil {
		return err
	}

	if _, err := h.RenewSession(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}

	h.Sessions.Set(ctx, sess.IsSignedIn, true)
	h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

	return nil
}

func SignInSetSession(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email string) error {
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
