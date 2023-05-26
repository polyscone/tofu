package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type RegenerateRecoveryCodesGuard interface {
	CanRegenerateRecoveryCodes(userID int) bool
}

func (s *Service) RegenerateRecoveryCodes(ctx context.Context, guard RegenerateRecoveryCodesGuard, userID int) error {
	var input struct {
		userID int
	}
	{
		if !guard.CanRegenerateRecoveryCodes(userID) {
			return errors.Tracef(app.ErrUnauthorised)
		}

		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.RegenerateRecoveryCodes(); err != nil {
		return errors.Tracef(err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
