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

type ChangePasswordGuard interface {
	CanChangePassword(userID uuid.V4) bool
}

type changePasswordData struct {
	userID      uuid.V4
	newPassword domain.Password
}

type ChangePassword struct {
	Guard       ChangePasswordGuard
	UserID      string
	NewPassword string
}

func (cmd ChangePassword) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return err
}

func (cmd ChangePassword) Validate(ctx context.Context) error {
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd ChangePassword) data(ctx context.Context) (changePasswordData, error) {
	var data changePasswordData
	if !cmd.Guard.CanChangePassword(uuid.ParseV4OrNil(cmd.UserID)) {
		return data, errors.Tracef(port.ErrUnauthorized, "insufficient permissions")
	}

	var err error
	var errs errors.Map

	if data.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if data.newPassword, err = domain.NewPassword(cmd.NewPassword); err != nil {
		errs.Set("new password", err)
	}

	return data, errs.Tracef(port.ErrInvalidInput)
}

type ChangePasswordHandler func(ctx context.Context, cmd ChangePassword) error

func NewChangePasswordHandler(broker event.Broker, users UserRepo) ChangePasswordHandler {
	return func(ctx context.Context, cmd ChangePassword) error {
		data, err := cmd.data(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, data.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ChangePassword(data.newPassword); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
