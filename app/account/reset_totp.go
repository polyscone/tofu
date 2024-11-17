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

type ResetTOTPInput struct {
	UserID   int
	Password Password
}

func (s *Service) ResetTOTPValidate(guard ResetTOTPGuard, userID int, password string) (ResetTOTPInput, error) {
	var input ResetTOTPInput

	if !guard.CanResetTOTP(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.UserID = userID

	if input.Password, err = NewPassword(password); err != nil {
		errs.Set("password", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) ResetTOTP(ctx context.Context, guard ResetTOTPGuard, userID int, password string) (*User, error) {
	input, err := s.ResetTOTPValidate(guard, userID, password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := user.ResetTOTP(input.Password, s.hasher); err != nil {
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
