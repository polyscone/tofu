package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type ActivateTOTPGuard interface {
	CanActivateTOTP(userID int) bool
}

type ActivateTOTPData struct {
	UserID int
}

func (s *Service) ActivateTOTPValidate(guard ActivateTOTPGuard, userID int) (ActivateTOTPData, error) {
	var data ActivateTOTPData

	if !guard.CanActivateTOTP(userID) {
		return data, app.ErrForbidden
	}

	data.UserID = userID

	return data, nil
}

func (s *Service) ActivateTOTP(ctx context.Context, guard ActivateTOTPGuard, userID int) (*User, error) {
	data, err := s.ActivateTOTPValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ActivateTOTP(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
