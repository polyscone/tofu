package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type RegenerateRecoveryCodesGuard interface {
	CanRegenerateRecoveryCodes(userID int) bool
}

func (s *Service) RegenerateRecoveryCodes(ctx context.Context, guard RegenerateRecoveryCodesGuard, userID int, totp string) error {
	var input struct {
		userID int
		totp   TOTP
	}
	{
		if !guard.CanRegenerateRecoveryCodes(userID) {
			return errors.Tracef(app.ErrUnauthorised)
		}

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

	if err := user.RegenerateRecoveryCodes(input.totp); err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
