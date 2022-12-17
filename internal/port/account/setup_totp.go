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

type setupTOTPRequest struct {
	userID uuid.V4
}

type setupTOTPResponse struct {
	Key []byte
}

type SetupTOTP struct {
	Guard  SetupTOTPGuard
	UserID string
}

func (cmd SetupTOTP) Execute(ctx context.Context, bus command.Bus) (setupTOTPResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(setupTOTPResponse), err
}

func (cmd SetupTOTP) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd SetupTOTP) request(ctx context.Context) (setupTOTPRequest, error) {
	var req setupTOTPRequest
	if !cmd.Guard.CanSetupTOTP(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type SetupTOTPHandler func(ctx context.Context, cmd SetupTOTP) (setupTOTPResponse, error)

func NewSetupTOTPHandler(broker event.Broker, users UserRepo) SetupTOTPHandler {
	return func(ctx context.Context, cmd SetupTOTP) (setupTOTPResponse, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return setupTOTPResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return setupTOTPResponse{}, errors.Tracef(err)
		}

		key, err := user.SetupTOTP()
		if err != nil {
			return setupTOTPResponse{}, errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return setupTOTPResponse{}, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		res := setupTOTPResponse{Key: key}

		return res, nil
	}
}
