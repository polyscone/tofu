package auth

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
)

func StartImpersonatingUser(ctx context.Context, h *handler.Handler, userID int) error {
	passport := h.Passport(ctx)
	if !passport.Account.CanImpersonateUser(userID) {
		return fmt.Errorf("%w: cannot impersonate user %v", app.ErrForbidden, userID)
	}

	imposter := h.User(ctx)
	if imposter.ID == userID {
		return nil
	}

	user, err := h.Repo.Account.FindUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	// We need to preserve the original imposter user id, so we
	// only set the imposter user id if it's not already set
	//
	// This avoids it being overwritten if the user impersonates
	// a user who can also impersonate users
	if h.Session.ImposterUserID(ctx) == 0 {
		h.Session.SetImposterUserID(ctx, imposter.ID)
	}

	// Re-query the imposter user to ensure we have the original
	// imposter user's data
	imposterID := h.Session.ImposterUserID(ctx)
	imposter, err = h.Repo.Account.FindUserByID(ctx, imposterID)
	if err != nil {
		return fmt.Errorf("find imposter user by id: %w", err)
	}

	h.Session.SetUserID(ctx, user.ID)
	h.Session.SetEmail(ctx, user.Email)

	h.Broker.Dispatch(ctx, event.UserImpersonationStarted{
		ImposterEmail: imposter.Email,
		UserEmail:     user.Email,
	})

	return nil
}

func StopImpersonatingUser(ctx context.Context, h *handler.Handler) error {
	imposterID := h.Session.ImposterUserID(ctx)
	if imposterID == 0 {
		return nil
	}

	user := h.User(ctx)

	imposter, err := h.Repo.Account.FindUserByID(ctx, imposterID)
	if err != nil {
		return fmt.Errorf("find imposter user by id: %w", err)
	}

	h.Session.DeleteImposterUserID(ctx)
	h.Session.SetUserID(ctx, imposter.ID)
	h.Session.SetEmail(ctx, imposter.Email)

	h.Broker.Dispatch(ctx, event.UserImpersonationStopped{
		ImposterEmail: imposter.Email,
		UserEmail:     user.Email,
	})

	return nil
}
