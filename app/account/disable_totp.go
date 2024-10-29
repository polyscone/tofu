package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type DisableTOTPGuard interface {
	CanDisableTOTP(userID int) bool
}

type DisableTOTPInput struct {
	UserID   int
	Password Password
}

func (s *Service) DisableTOTPValidate(guard DisableTOTPGuard, userID int, password string) (DisableTOTPInput, error) {
	var input DisableTOTPInput

	if !guard.CanDisableTOTP(userID) {
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

func (s *Service) DisableTOTP(ctx context.Context, guard DisableTOTPGuard, userID int, password string) (*User, error) {
	input, err := s.DisableTOTPValidate(guard, userID, password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.DisableTOTP(input.Password, s.hasher); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return nil, fmt.Errorf("%w: %w", app.ErrUnauthorised, err)
		}

		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
