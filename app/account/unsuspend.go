package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type UnsuspendUsersGuard interface {
	CanUnsuspendUsers() bool
}

func (s *Service) UnsuspendUser(ctx context.Context, guard UnsuspendUsersGuard, userID int) (*User, error) {
	var input struct {
		userID int
	}
	{
		if !guard.CanUnsuspendUsers() {
			return nil, app.ErrForbidden
		}

		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.Unsuspend()

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
