package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type VerifyTOTPGuard interface {
	CanVerifyTOTP(userID int) bool
}

type VerifyTOTPInput struct {
	UserID     int
	TOTP       TOTP
	TOTPMethod TOTPMethod
}

func (s *Service) VerifyTOTPValidate(guard VerifyTOTPGuard, userID int, totp, totpMethod string) (VerifyTOTPInput, error) {
	var input VerifyTOTPInput

	if !guard.CanVerifyTOTP(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.UserID = userID

	if input.TOTP, err = NewTOTP(totp); err != nil {
		errs.Set("totp", err)
	}
	if input.TOTPMethod, err = NewTOTPMethod(totpMethod); err != nil {
		errs.Set("totp kind", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) VerifyTOTP(ctx context.Context, guard VerifyTOTPGuard, userID int, totp, totpMethod string) (*User, []string, error) {
	input, err := s.VerifyTOTPValidate(guard, userID, totp, totpMethod)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("find user by id: %w", err)
	}

	codes, err := user.VerifyTOTP(input.TOTP, input.TOTPMethod)
	if err != nil {
		return nil, nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, codes, nil
}
