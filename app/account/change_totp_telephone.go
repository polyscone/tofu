package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
)

type ChangeTOTPTelGuard interface {
	CanChangeTOTPTel(userID string) bool
}

func (s *Service) ChangeTOTPTel(ctx context.Context, guard ChangeTOTPTelGuard, userID string, newTel string) error {
	var input struct {
		userID UserID
		newTel Tel
	}
	{
		if !guard.CanChangeTOTPTel(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
		}
		if input.newTel, err = NewTel(newTel); err != nil {
			errs.Set("new phone", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
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
