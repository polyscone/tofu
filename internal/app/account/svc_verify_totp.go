package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type VerifyTOTPGuard interface {
	CanVerifyTOTP(userID int) bool
}

func (s *Service) VerifyTOTP(ctx context.Context, guard VerifyTOTPGuard, userID int, totp, totpMethod string) error {
	var input struct {
		userID     int
		totp       TOTP
		totpMethod TOTPMethod
	}
	{
		if !guard.CanVerifyTOTP(userID) {
			return app.ErrUnauthorised
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}
		if input.totpMethod, err = NewTOTPMethod(totpMethod); err != nil {
			errs.Set("totp kind", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.VerifyTOTP(input.totp, input.totpMethod); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
