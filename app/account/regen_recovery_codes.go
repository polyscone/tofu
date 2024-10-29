package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type RegenerateRecoveryCodesGuard interface {
	CanRegenerateRecoveryCodes(userID int) bool
}

type RegenerateRecoveryCodesInput struct {
	UserID int
	TOTP   TOTP
}

func (s *Service) RegenerateRecoveryCodesValidate(guard RegenerateRecoveryCodesGuard, userID int, totp string) (RegenerateRecoveryCodesInput, error) {
	var input RegenerateRecoveryCodesInput

	if !guard.CanRegenerateRecoveryCodes(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.UserID = userID

	if input.TOTP, err = NewTOTP(totp); err != nil {
		errs.Set("totp", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) RegenerateRecoveryCodes(ctx context.Context, guard RegenerateRecoveryCodesGuard, userID int, totp string) (*User, []string, error) {
	input, err := s.RegenerateRecoveryCodesValidate(guard, userID, totp)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("find user by id: %w", err)
	}

	codes, err := user.RegenerateRecoveryCodes(input.TOTP)
	if err != nil {
		return nil, nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, codes, nil
}
