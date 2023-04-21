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

type changePasswordRequest struct {
	userID           uuid.V4
	oldPassword      domain.Password
	newPassword      domain.Password
	newPasswordCheck domain.Password
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

func (cmd ChangePassword) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd ChangePassword) request(ctx context.Context) (changePasswordRequest, error) {
	var req changePasswordRequest
	if !cmd.Guard.CanChangePassword(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.oldPassword, err = domain.NewPassword(cmd.OldPassword); err != nil {
		errs.Set("old password", err)
	}
	if req.newPassword, err = domain.NewPassword(cmd.NewPassword); err != nil {
		errs.Set("new password", err)
	}
	if req.newPasswordCheck, err = domain.NewPassword(cmd.NewPasswordCheck); err != nil {
		errs.Set("new password check", err)
	}
	if !req.newPassword.Equal(req.newPasswordCheck) {
		errs.Set("new password", "passwords do not match")
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type ChangePasswordHandler func(ctx context.Context, cmd ChangePassword) error

func NewChangePasswordHandler(broker event.Broker, users UserRepo) ChangePasswordHandler {
	return func(ctx context.Context, cmd ChangePassword) error {
		req, err := cmd.request(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ChangePassword(req.oldPassword, req.newPassword); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
