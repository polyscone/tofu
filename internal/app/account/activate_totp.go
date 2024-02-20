package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/uuid"
)

type ActivateTOTPGuard interface {
	CanActivateTOTP(userID string) bool
}

func (s *Service) ActivateTOTP(ctx context.Context, guard ActivateTOTPGuard, userID string) error {
	var input struct {
		userID uuid.UUID
	}
	{
		if !guard.CanActivateTOTP(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = uuid.Parse(userID); err != nil {
			errs.Set("user id", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ActivateTOTP(); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
