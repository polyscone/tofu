package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ResetTOTPGuard interface {
	CanResetTOTP(userID int) bool
}

type ResetTOTPData struct {
	UserID   int
	Password Password
}

func (s *Service) ResetTOTPValidate(guard ResetTOTPGuard, userID int, password string) (ResetTOTPData, error) {
	var data ResetTOTPData

	if !guard.CanResetTOTP(userID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.Password, err = NewPassword(password); err != nil {
		errs.Set("password", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) ResetTOTP(ctx context.Context, guard ResetTOTPGuard, userID int, password string) (*User, error) {
	data, err := s.ResetTOTPValidate(guard, userID, password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := user.ResetTOTP(data.Password, s.hasher); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return nil, fmt.Errorf("%w: %w", app.ErrUnauthorized, err)
		}

		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
