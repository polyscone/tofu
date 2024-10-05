package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type UnsuspendUsersGuard interface {
	CanUnsuspendUsers() bool
}

type UnsuspendUserInput struct {
	UserID int
}

func (s *Service) UnsuspendUserValidate(guard UnsuspendUsersGuard, userID int) (UnsuspendUserInput, error) {
	var input UnsuspendUserInput

	if !guard.CanUnsuspendUsers() {
		return input, app.ErrForbidden
	}

	input.UserID = userID

	return input, nil
}

func (s *Service) UnsuspendUser(ctx context.Context, guard UnsuspendUsersGuard, userID int) (*User, error) {
	input, err := s.UnsuspendUserValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.Unsuspend()

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
