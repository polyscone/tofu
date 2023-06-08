package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type ChangeTOTPTelephoneGuard interface {
	CanChangeTOTPTelephone(userID int) bool
}

func (s *Service) ChangeTOTPTelephone(ctx context.Context, guard ChangeTOTPTelephoneGuard, userID int, newTelephone string) error {
	var input struct {
		userID       int
		newTelephone text.Tel
	}
	{
		if !guard.CanChangeTOTPTelephone(userID) {
			return errors.Tracef(app.ErrUnauthorised)
		}

		var err error
		var errs errors.Map

		input.userID = userID

		if input.newTelephone, err = text.NewTel(newTelephone); err != nil {
			errs.Set("new telephone", err)
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.store.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.ChangeTOTPTelephone(input.newTelephone); err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
