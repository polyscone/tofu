package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type ResetTOTPGuard interface {
	CanResetTOTP(userID int) bool
}

func (s *Service) ResetTOTP(ctx context.Context, guard ResetTOTPGuard, userID int, password string) (*User, error) {
	var input struct {
		userID   int
		password Password
	}
	{
		if !guard.CanResetTOTP(userID) {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := user.ResetTOTP(input.password, s.hasher); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return nil, fmt.Errorf("%w: %w", app.ErrUnauthorised, err)
		}

		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
