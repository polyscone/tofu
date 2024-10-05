package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type ActivateTOTPGuard interface {
	CanActivateTOTP(userID int) bool
}

type ActivateTOTPInput struct {
	UserID int
}

func (s *Service) ActivateTOTPValidate(guard ActivateTOTPGuard, userID int) (ActivateTOTPInput, error) {
	var input ActivateTOTPInput

	if !guard.CanActivateTOTP(userID) {
		return input, app.ErrForbidden
	}

	input.UserID = userID

	return input, nil
}

func (s *Service) ActivateTOTP(ctx context.Context, guard ActivateTOTPGuard, userID int) (*User, error) {
	input, err := s.ActivateTOTPValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
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
