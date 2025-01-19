package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type SetupTOTPGuard interface {
	CanSetupTOTP(userID int) bool
}

type SetupTOTPData struct {
	UserID int
}

func (s *Service) SetupTOTPValidate(guard SetupTOTPGuard, userID int) (SetupTOTPData, error) {
	var data SetupTOTPData

	if !guard.CanSetupTOTP(userID) {
		return data, app.ErrForbidden
	}

	data.UserID = userID

	return data, nil
}

func (s *Service) SetupTOTP(ctx context.Context, guard SetupTOTPGuard, userID int) (*User, error) {
	data, err := s.SetupTOTPValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
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
