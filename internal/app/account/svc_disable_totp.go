package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
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
			return errors.Tracef(app.ErrUnauthorised)
		}

		var err error
		var errs errors.Map

		input.userID = userID

		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.DisableTOTP(input.password, s.hasher); err != nil {
		return errors.Tracef(err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}