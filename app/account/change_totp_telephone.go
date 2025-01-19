package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ChangeTOTPTelGuard interface {
	CanChangeTOTPTel(userID int) bool
}

type ChangeTOTPTelData struct {
	UserID int
	NewTel Tel
}

func (s *Service) ChangeTOTPTelValidate(guard ChangeTOTPTelGuard, userID int, newTel string) (ChangeTOTPTelData, error) {
	var data ChangeTOTPTelData

	if !guard.CanChangeTOTPTel(userID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.NewTel, err = NewTel(newTel); err != nil {
		errs.Set("new phone", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) ChangeTOTPTel(ctx context.Context, guard ChangeTOTPTelGuard, userID int, newTel string) (*User, error) {
	data, err := s.ChangeTOTPTelValidate(guard, userID, newTel)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChangeTOTPTel(data.NewTel); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
