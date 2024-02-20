package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/uuid"
)

func (s *Service) SignInWithTOTP(ctx context.Context, userID, totp string) error {
	var input struct {
		userID uuid.UUID
		totp   TOTP
	}
	{
		var err error
		var errs errsx.Map

		if input.userID, err = uuid.Parse(userID); err != nil {
			errs.Set("user id", err)
		}
		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithTOTP(s.system, input.totp); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
