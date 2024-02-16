package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type ResetTOTPGuard interface {
	CanResetTOTP(userID int) bool
}

func (s *Service) ResetTOTP(ctx context.Context, guard ResetTOTPGuard, userID int, password string) error {
	var input struct {
		userID   int
		password Password
	}
	{
		if !guard.CanResetTOTP(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	if err := user.ResetTOTP(input.password, s.hasher); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return fmt.Errorf("%w: %w", app.ErrUnauthorised, err)
		}

		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
