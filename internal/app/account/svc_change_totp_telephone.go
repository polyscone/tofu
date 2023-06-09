package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type ChangeTOTPTelGuard interface {
	CanChangeTOTPTel(userID int) bool
}

func (s *Service) ChangeTOTPTel(ctx context.Context, guard ChangeTOTPTelGuard, userID int, newTel string) error {
	var input struct {
		userID int
		newTel Tel
	}
	{
		if !guard.CanChangeTOTPTel(userID) {
			return errors.Tracef(app.ErrUnauthorised)
		}

		var err error
		var errs errors.Map

		input.userID = userID

		if input.newTel, err = NewTel(newTel); err != nil {
			errs.Set("new phone", err)
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.store.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.ChangeTOTPTel(input.newTel); err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
