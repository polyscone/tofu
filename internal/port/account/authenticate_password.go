package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

type authenticateWithPasswordRequest struct {
	email    text.Email
	password domain.Password
}

type AuthenticateWithPassword struct {
	Email    string
	Password string
}

func (cmd AuthenticateWithPassword) Execute(ctx context.Context, bus command.Bus) (Passport, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(Passport), errors.Tracef(err)
}

func (cmd AuthenticateWithPassword) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithPassword) request(ctx context.Context) (authenticateWithPasswordRequest, error) {
	var req authenticateWithPasswordRequest
	var err error
	var errs errors.Map

	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}
	if req.password, err = domain.NewPassword(cmd.Password); err != nil {
		errs.Set("password", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type AuthenticateWithPasswordHandler func(ctx context.Context, cmd AuthenticateWithPassword) (Passport, error)

func NewAuthenticateWithPasswordHandler(broker event.Broker, users UserRepo) AuthenticateWithPasswordHandler {
	return func(ctx context.Context, cmd AuthenticateWithPassword) (Passport, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, req.email)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		if err := user.AuthenticateWithPassword(req.password); err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return newPassport(user), nil
	}
}