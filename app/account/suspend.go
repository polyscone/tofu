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

type SuspendUserData struct {
	UserID          int
	SuspendedReason SuspendedReason
}

func (s *Service) SuspendUserValidate(guard SuspendUsersGuard, userID int, suspendedReason string) (SuspendUserData, error) {
	var data SuspendUserData

	if !guard.CanSuspendUsers() {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.SuspendedReason, err = NewSuspendedReason(suspendedReason); err != nil {
		errs.Set("suspended reason", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) SuspendUser(ctx context.Context, guard SuspendUsersGuard, userID int, suspendedReason string) (*User, error) {
	data, err := s.SuspendUserValidate(guard, userID, suspendedReason)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.Suspend(data.SuspendedReason)

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
