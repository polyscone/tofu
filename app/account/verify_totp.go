package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
)

type VerifyTOTPGuard interface {
	CanVerifyTOTP(userID string) bool
}

func (s *Service) VerifyTOTP(ctx context.Context, guard VerifyTOTPGuard, userID, totp, totpMethod string) ([]string, error) {
	var input struct {
		userID     UserID
		totp       TOTP
		totpMethod TOTPMethod
	}
	{
		if !guard.CanVerifyTOTP(userID) {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
		}
		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}
		if input.totpMethod, err = NewTOTPMethod(totpMethod); err != nil {
			errs.Set("totp kind", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	codes, err := user.VerifyTOTP(input.totp, input.totpMethod)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return codes, nil
}
