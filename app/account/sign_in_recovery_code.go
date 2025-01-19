package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type SignInWithRecoveryCodeData struct {
	UserID       int
	RecoveryCode RecoveryCode
}

func (s *Service) SignInWithRecoveryCodeValidate(userID int, recoveryCode string) (SignInWithRecoveryCodeData, error) {
	var data SignInWithRecoveryCodeData
	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.RecoveryCode, err = NewRecoveryCode(recoveryCode); err != nil {
		errs.Set("recovery code", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) signInWithRecoveryCode(ctx context.Context, userID int, recoveryCode string) (*User, error) {
	data, err := s.SignInWithRecoveryCodeValidate(userID, recoveryCode)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithRecoveryCode(s.system, data.RecoveryCode); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}

func (s *Service) SignInWithRecoveryCode(ctx context.Context, userID int, recoveryCode string) (*User, error) {
	user, err := s.signInWithRecoveryCode(ctx, userID, recoveryCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
