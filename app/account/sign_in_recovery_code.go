package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

func (s *Service) signInWithRecoveryCode(ctx context.Context, userID int, recoveryCode string) (*User, error) {
	var input struct {
		userID       int
		recoveryCode RecoveryCode
	}
	{
		var err error
		var errs errsx.Map

		input.userID = userID

		if input.recoveryCode, err = NewRecoveryCode(recoveryCode); err != nil {
			errs.Set("recovery code", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithRecoveryCode(s.system, input.recoveryCode); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}

func (s *Service) SignInWithRecoveryCode(ctx context.Context, userID int, recoveryCode string) (*User, error) {
	user, err := s.signInWithRecoveryCode(ctx, userID, recoveryCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
