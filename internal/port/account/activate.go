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

type activateRequest struct {
	email         text.Email
	password      domain.Password
	passwordCheck domain.Password
}

type Activate struct {
	Email         string
	Password      string
	PasswordCheck string
}

func (cmd Activate) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd Activate) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd Activate) request(ctx context.Context) (activateRequest, error) {
	var req activateRequest
	var err error
	var errs errors.Map

	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}
	if req.password, err = domain.NewPassword(cmd.Password); err != nil {
		errs.Set("password", err)
	}
	if req.passwordCheck, err = domain.NewPassword(cmd.PasswordCheck); err != nil {
		errs.Set("password check", err)
	}
	if !req.password.Equal(req.passwordCheck) {
		errs.Set("password", "passwords do not match")
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type ActivateHandler func(ctx context.Context, cmd Activate) error

func NewActivateHandler(broker event.Broker, users UserRepo) ActivateHandler {
	return func(ctx context.Context, cmd Activate) error {
		req, err := cmd.request(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, req.email)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ActivateAndSetPassword(req.password); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
