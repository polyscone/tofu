package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/domain"
)

type authenticateWithPasswordRequest struct {
	email    text.Email
	password domain.Password
}

type authenticateWithPasswordResponse struct {
	UserID          string
	HasVerifiedTOTP bool
	TOTPUseSMS      bool
}

type AuthenticateWithPassword struct {
	Email    string
	Password string
}

func (cmd AuthenticateWithPassword) Execute(ctx context.Context, bus command.Bus) (authenticateWithPasswordResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(authenticateWithPasswordResponse), errors.Tracef(err)
}

func (cmd AuthenticateWithPassword) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd AuthenticateWithPassword) request() (authenticateWithPasswordRequest, error) {
	var req authenticateWithPasswordRequest
	var err error
	var errs errors.Map

	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}
	if req.password, err = domain.NewPassword(cmd.Password); err != nil {
		errs.Set("password", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type AuthenticateWithPasswordHandler func(ctx context.Context, cmd AuthenticateWithPassword) (authenticateWithPasswordResponse, error)

func NewAuthenticateWithPasswordHandler(broker event.Broker, users UserRepo) AuthenticateWithPasswordHandler {
	return func(ctx context.Context, cmd AuthenticateWithPassword) (authenticateWithPasswordResponse, error) {
		req, err := cmd.request()
		if err != nil {
			return authenticateWithPasswordResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, req.email)
		if err != nil {
			return authenticateWithPasswordResponse{}, errors.Tracef(err)
		}

		if err := user.AuthenticateWithPassword(req.password); err != nil {
			return authenticateWithPasswordResponse{}, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		res := authenticateWithPasswordResponse{
			UserID:          user.ID.String(),
			HasVerifiedTOTP: user.HasVerifiedTOTP(),
			TOTPUseSMS:      user.TOTPUseSMS,
		}

		return res, nil
	}
}
