package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type RequestTOTPResetData struct {
	Email Email
}

func (s *Service) RequestTOTPResetValidate(email string) (RequestTOTPResetData, error) {
	var data RequestTOTPResetData
	var err error
	var errs errsx.Map

	if data.Email, err = NewEmail(email); err != nil {
		errs.Set("email", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) RequestTOTPReset(ctx context.Context, email string) (*User, error) {
	data, err := s.RequestTOTPResetValidate(email)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByEmail(ctx, data.Email.String())
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := user.RequestTOTPReset(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
