package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func (s *Service) SignInWithTOTP(ctx context.Context, userID int, totp string) error {
	var input struct {
		userID int
		totp   TOTP
	}
	{
		var err error
		var errs errors.Map

		input.userID = userID

		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.store.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.SignInWithTOTP(input.totp); err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
