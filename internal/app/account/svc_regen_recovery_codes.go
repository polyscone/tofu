package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
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
			return app.ErrUnauthorised
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.RegenerateRecoveryCodes(input.totp); err != nil {
		return fmt.Errorf("regenerate recovery codes: %w", err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
