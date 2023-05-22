package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type registerRequest struct {
	userID   uuid.V4
	email    text.Email
	password Password
}

type Register struct {
	UserID        string
	Email         string
	Password      string
	PasswordCheck string
}

func (cmd Register) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd Register) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd Register) request() (registerRequest, error) {
	var req registerRequest
	var err error
	var errs errors.Map

	passwordCheck, _ := NewPassword(cmd.PasswordCheck)

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("id", err)
	}
	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}
	if req.password, err = NewPassword(cmd.Password); err != nil {
		errs.Set("password", err)
	} else if !req.password.Equal(passwordCheck) {
		errs.Set("password", "passwords do not match")
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type RegisterHandler func(ctx context.Context, cmd Register) error

func NewRegisterHandler(broker event.Broker, hasher password.Hasher, users UserRepo) RegisterHandler {
	return func(ctx context.Context, cmd Register) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user := NewUser(req.userID)

		if err := user.Register(req.email, req.password, hasher); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Add(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
