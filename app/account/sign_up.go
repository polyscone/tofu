package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type SignUpData struct {
	Email Email
}

func (s *Service) SignUpValidate(email string) (SignUpData, error) {
	var data SignUpData
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

func (s *Service) SignUp(ctx context.Context, email string) (*User, error) {
	data, err := s.SignUpValidate(email)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByEmail(ctx, data.Email.String())
	switch {
	case err == nil:
		user.SignUp(s.system)

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return nil, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		user = NewUser(data.Email)

		user.SignUp(s.system)

		if err := s.repo.AddUser(ctx, user); err != nil {
			return nil, fmt.Errorf("add user: %w", err)
		}

	default:
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
