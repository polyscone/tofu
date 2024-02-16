package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
)

type SetupTOTPGuard interface {
	CanSetupTOTP(userID int) bool
}

func (s *Service) SetupTOTP(ctx context.Context, guard SetupTOTPGuard, userID int) error {
	var input struct {
		userID int
	}
	{
		if !guard.CanSetupTOTP(userID) {
			return app.ErrForbidden
		}

		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SetupTOTP(); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
