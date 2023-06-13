package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
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
			return app.ErrUnauthorised
		}

		var err error
		var errs errsx.Map

		input.userID = userID

		if input.newTel, err = NewTel(newTel); err != nil {
			errs.Set("new phone", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChangeTOTPTel(input.newTel); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
