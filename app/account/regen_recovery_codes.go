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

type RegenerateRecoveryCodesData struct {
	UserID int
	TOTP   TOTP
}

func (s *Service) RegenerateRecoveryCodesValidate(guard RegenerateRecoveryCodesGuard, userID int, totp string) (RegenerateRecoveryCodesData, error) {
	var data RegenerateRecoveryCodesData

	if !guard.CanRegenerateRecoveryCodes(userID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.TOTP, err = NewTOTP(totp); err != nil {
		errs.Set("totp", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) RegenerateRecoveryCodes(ctx context.Context, guard RegenerateRecoveryCodesGuard, userID int, totp string) (*User, []string, error) {
	data, err := s.RegenerateRecoveryCodesValidate(guard, userID, totp)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("find user by id: %w", err)
	}

	codes, err := user.RegenerateRecoveryCodes(data.TOTP)
	if err != nil {
		return nil, nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, codes, nil
}
