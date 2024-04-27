package account

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
)

type ActivateUsersGuard interface {
	CanActivateUsers() bool
}

func (s *Service) ActivateUser(ctx context.Context, guard ActivateUsersGuard, userID string) error {
	var input struct {
		userID UserID
	}
	{
		if !guard.CanActivateUsers() {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	if err := user.Activate(); err != nil {
		return err
	}

	log.Attempts = 0
	log.LastAttemptAt = time.Time{}

	if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
		return fmt.Errorf("save sign in attempt log: %w", err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
