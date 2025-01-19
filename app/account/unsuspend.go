package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type UnsuspendUsersGuard interface {
	CanUnsuspendUsers() bool
}

type UnsuspendUserData struct {
	UserID int
}

func (s *Service) UnsuspendUserValidate(guard UnsuspendUsersGuard, userID int) (UnsuspendUserData, error) {
	var data UnsuspendUserData

	if !guard.CanUnsuspendUsers() {
		return data, app.ErrForbidden
	}

	data.UserID = userID

	return data, nil
}

func (s *Service) UnsuspendUser(ctx context.Context, guard UnsuspendUsersGuard, userID int) (*User, error) {
	data, err := s.UnsuspendUserValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	user.Unsuspend()

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
