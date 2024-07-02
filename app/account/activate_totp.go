package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type ActivateTOTPGuard interface {
	CanActivateTOTP(userID int) bool
}

func (s *Service) ActivateTOTP(ctx context.Context, guard ActivateTOTPGuard, userID int) (*User, error) {
	var input struct {
		userID int
	}
	{
		if !guard.CanActivateTOTP(userID) {
			return nil, app.ErrForbidden
		}

		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ActivateTOTP(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
