package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
)

type UnsuspendUsersGuard interface {
	CanUnsuspendUsers() bool
}

func (s *Service) UnsuspendUser(ctx context.Context, guard UnsuspendUsersGuard, userID int) error {
	var input struct {
		userID int
	}
	{
		if !guard.CanUnsuspendUsers() {
			return app.ErrUnauthorised
		}

		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	user.Unsuspend()

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
