package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

type ResetPasswordGuard interface {
	CanResetPassword(userID uuid.V4) bool
}

type resetPasswordRequest struct {
	userID      uuid.V4
	newPassword domain.Password
}

type ResetPassword struct {
	Guard       ResetPasswordGuard
	UserID      string
	NewPassword string
}

func (cmd ResetPassword) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd ResetPassword) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd ResetPassword) request(ctx context.Context) (resetPasswordRequest, error) {
	var req resetPasswordRequest
	if !cmd.Guard.CanResetPassword(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.newPassword, err = domain.NewPassword(cmd.NewPassword); err != nil {
		errs.Set("new password", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type ResetPasswordHandler func(ctx context.Context, cmd ResetPassword) error

func NewResetPasswordHandler(broker event.Broker, users UserRepo) ResetPasswordHandler {
	return func(ctx context.Context, cmd ResetPassword) error {
		req, err := cmd.request(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ResetPassword(req.newPassword); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
