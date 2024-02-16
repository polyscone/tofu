package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func (s *Service) SignInWithRecoveryCode(ctx context.Context, userID int, recoveryCode string) error {
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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithRecoveryCode(s.system, input.recoveryCode); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
