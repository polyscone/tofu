package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type SetupTOTPGuard interface {
	CanSetupTOTP(userID int) bool
}

type SetupTOTPInput struct {
	UserID int
}

func (s *Service) SetupTOTPValidate(guard SetupTOTPGuard, userID int) (SetupTOTPInput, error) {
	var input SetupTOTPInput

	if !guard.CanSetupTOTP(userID) {
		return input, app.ErrForbidden
	}

	input.UserID = userID

	return input, nil
}

func (s *Service) SetupTOTP(ctx context.Context, guard SetupTOTPGuard, userID int) (*User, error) {
	input, err := s.SetupTOTPValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SetupTOTP(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
