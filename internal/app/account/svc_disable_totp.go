package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type DisableTOTPGuard interface {
	CanDisableTOTP(userID int) bool
}

func (s *Service) DisableTOTP(ctx context.Context, guard DisableTOTPGuard, userID int, password string) error {
	var input struct {
		userID   int
		password Password
	}
	{
		if !guard.CanDisableTOTP(userID) {
			return app.ErrUnauthorised
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

	user, err := s.store.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.DisableTOTP(input.password, s.hasher); err != nil {
		return fmt.Errorf("disable TOTP: %w", err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
