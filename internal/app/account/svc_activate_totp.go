package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type ActivateTOTPGuard interface {
	CanActivateTOTP(userID int) bool
}

func (s *Service) ActivateTOTP(ctx context.Context, guard ActivateTOTPGuard, userID int) error {
	var input struct {
		userID int
	}
	{
		if !guard.CanActivateTOTP(userID) {
			return errors.Tracef(app.ErrUnauthorised)
		}

		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.ActivateTOTP(); err != nil {
		return errors.Tracef(err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
