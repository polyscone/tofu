package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type DisableTOTPGuard interface {
	CanDisableTOTP(userID string) bool
}

func (s *Service) DisableTOTP(ctx context.Context, guard DisableTOTPGuard, userID, password string) error {
	var input struct {
		userID   ID
		password Password
	}
	{
		if !guard.CanDisableTOTP(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseID(userID); err != nil {
			errs.Set("user id", err)
		}
		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.DisableTOTP(input.password, s.hasher); err != nil {
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
