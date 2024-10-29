package account

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
)

type ActivateUsersGuard interface {
	CanActivateUsers() bool
}

type ActivateUserInput struct {
	UserID int
}

func (s *Service) ActivateUserValidate(guard ActivateUsersGuard, userID int) (ActivateUserInput, error) {
	var input ActivateUserInput

	if !guard.CanActivateUsers() {
		return input, app.ErrForbidden
	}

	input.UserID = userID

	return input, nil
}

func (s *Service) ActivateUser(ctx context.Context, guard ActivateUsersGuard, userID int) (*User, error) {
	input, err := s.ActivateUserValidate(guard, userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, user.Email)
	if err != nil {
		return nil, fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	if err := user.Activate(); err != nil {
		return nil, err
	}

	log.Attempts = 0
	log.LastAttemptAt = time.Time{}

	if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
		return nil, fmt.Errorf("save sign in attempt log: %w", err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
