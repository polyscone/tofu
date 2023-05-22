package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type ChangePasswordGuard interface {
	CanChangePassword(userID uuid.V4) bool
}

type changePasswordRequest struct {
	userID      uuid.V4
	oldPassword Password
	newPassword Password
}

type ChangePassword struct {
	Guard            ChangePasswordGuard
	UserID           string
	OldPassword      string
	NewPassword      string
	NewPasswordCheck string
}

func (cmd ChangePassword) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd ChangePassword) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd ChangePassword) request() (changePasswordRequest, error) {
	var req changePasswordRequest
	if !cmd.Guard.CanChangePassword(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	newPasswordCheck, _ := NewPassword(cmd.NewPasswordCheck)

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.oldPassword, err = NewPassword(cmd.OldPassword); err != nil {
		errs.Set("old password", err)
	}
	if req.newPassword, err = NewPassword(cmd.NewPassword); err != nil {
		errs.Set("new password", err)
	} else if !req.newPassword.Equal(newPasswordCheck) {
		errs.Set("new password", "passwords do not match")
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type ChangePasswordHandler func(ctx context.Context, cmd ChangePassword) error

func NewChangePasswordHandler(broker event.Broker, hasher password.Hasher, users UserRepo) ChangePasswordHandler {
	return func(ctx context.Context, cmd ChangePassword) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ChangePassword(req.oldPassword, req.newPassword, hasher); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
