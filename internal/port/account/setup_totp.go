package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type SetupTOTPGuard interface {
	CanSetupTOTP(userID uuid.V4) bool
}

type setupTOTPData struct {
	userID uuid.V4
}

type SetupTOTPResponse struct {
	Key []byte
}

type SetupTOTP struct {
	Guard  SetupTOTPGuard
	UserID string
}

func (cmd SetupTOTP) Execute(ctx context.Context, bus command.Bus) (SetupTOTPResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(SetupTOTPResponse), err
}

func (cmd SetupTOTP) Validate(ctx context.Context) error {
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd SetupTOTP) data(ctx context.Context) (setupTOTPData, error) {
	var data setupTOTPData
	if !cmd.Guard.CanSetupTOTP(uuid.ParseV4OrNil(cmd.UserID)) {
		return data, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if data.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	return data, errs.Tracef(port.ErrInvalidInput)
}

type SetupTOTPHandler func(ctx context.Context, cmd SetupTOTP) (SetupTOTPResponse, error)

func NewSetupTOTPHandler(broker event.Broker, users UserRepo) SetupTOTPHandler {
	return func(ctx context.Context, cmd SetupTOTP) (SetupTOTPResponse, error) {
		var res SetupTOTPResponse

		data, err := cmd.data(ctx)
		if err != nil {
			return res, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, data.userID)
		if err != nil {
			return res, errors.Tracef(err)
		}

		key, err := user.SetupTOTP()
		if err != nil {
			return res, errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return res, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		res = SetupTOTPResponse{Key: key}

		return res, nil
	}
}
