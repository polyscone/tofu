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
	TOTP       string
	TOTPMethod string
}

type VerifyTOTPData struct {
	UserID     int
	TOTP       TOTP
	TOTPMethod TOTPMethod
}

func (s *Service) VerifyTOTPValidate(guard VerifyTOTPGuard, input VerifyTOTPInput) (VerifyTOTPData, error) {
	var data VerifyTOTPData

	if !guard.CanVerifyTOTP(input.UserID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = input.UserID

	if data.TOTP, err = NewTOTP(input.TOTP); err != nil {
		errs.Set("totp", err)
	}
	if data.TOTPMethod, err = NewTOTPMethod(input.TOTPMethod); err != nil {
		errs.Set("totp kind", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) VerifyTOTP(ctx context.Context, guard VerifyTOTPGuard, input VerifyTOTPInput) (*User, []string, error) {
	data, err := s.VerifyTOTPValidate(guard, input)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("find user by id: %w", err)
	}

	codes, err := user.VerifyTOTP(data.TOTP, data.TOTPMethod)
	if err != nil {
		return nil, nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, codes, nil
}
