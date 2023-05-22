package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port"
)

type activateRequest struct {
	email text.Email
}

type Activate struct {
	Email string
}

func (cmd Activate) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd Activate) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd Activate) request() (activateRequest, error) {
	var req activateRequest
	var err error
	var errs errors.Map

	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type ActivateHandler func(ctx context.Context, cmd Activate) error

func NewActivateHandler(broker event.Broker, users UserRepo) ActivateHandler {
	return func(ctx context.Context, cmd Activate) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, req.email)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.Activate(); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
