package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type SuspendUsersGuard interface {
	CanSuspendUsers() bool
}

func (s *Service) SuspendUser(ctx context.Context, guard SuspendUsersGuard, userID, suspendedReason string) error {
	var input struct {
		userID          UserID
		suspendedReason SuspendedReason
	}
	{
		if !guard.CanSuspendUsers() {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
		}
		if input.suspendedReason, err = NewSuspendedReason(suspendedReason); err != nil {
			errs.Set("suspended reason", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.Suspend(input.suspendedReason); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
