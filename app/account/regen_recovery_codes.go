package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type RegenerateRecoveryCodesGuard interface {
	CanRegenerateRecoveryCodes(userID int) bool
}

func (s *Service) RegenerateRecoveryCodes(ctx context.Context, guard RegenerateRecoveryCodesGuard, userID int, totp string) ([]string, error) {
	var input struct {
		userID int
		totp   TOTP
	}
	{
		if !guard.CanRegenerateRecoveryCodes(userID) {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	codes, err := user.RegenerateRecoveryCodes(input.totp)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return codes, nil
}
