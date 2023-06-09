package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func (s *Service) SignInWithRecoveryCode(ctx context.Context, userID int, code string) error {
	var input struct {
		userID int
		code   RecoveryCode
	}
	{
		var err error
		var errs errors.Map

		input.userID = userID

		if input.code, err = NewRecoveryCode(code); err != nil {
			errs.Set("recovery code", err)
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.store.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.SignInWithRecoveryCode(input.code); err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
