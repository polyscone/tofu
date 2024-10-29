package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type SuspendUsersGuard interface {
	CanSuspendUsers() bool
}

type SuspendUserInput struct {
	UserID          int
	SuspendedReason SuspendedReason
}

func (s *Service) SuspendUserValidate(guard SuspendUsersGuard, userID int, suspendedReason string) (SuspendUserInput, error) {
	var input SuspendUserInput

	if !guard.CanSuspendUsers() {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.UserID = userID

	if input.SuspendedReason, err = NewSuspendedReason(suspendedReason); err != nil {
		errs.Set("suspended reason", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) SuspendUser(ctx context.Context, guard SuspendUsersGuard, userID int, suspendedReason string) (*User, error) {
	input, err := s.SuspendUserValidate(guard, userID, suspendedReason)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.Suspend(input.SuspendedReason)

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
