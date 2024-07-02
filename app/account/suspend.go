package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type SuspendUsersGuard interface {
	CanSuspendUsers() bool
}

func (s *Service) SuspendUser(ctx context.Context, guard SuspendUsersGuard, userID int, suspendedReason string) (*User, error) {
	var input struct {
		userID          int
		suspendedReason SuspendedReason
	}
	{
		if !guard.CanSuspendUsers() {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.suspendedReason, err = NewSuspendedReason(suspendedReason); err != nil {
			errs.Set("suspended reason", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.Suspend(input.suspendedReason)

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
