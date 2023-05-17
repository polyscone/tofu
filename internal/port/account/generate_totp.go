package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type GenerateTOTPGuard interface {
	CanGenerateTOTP(userID uuid.V4) bool
}

type generateTOTPRequest struct {
	userID uuid.V4
}

type generateTOTPResponse struct {
	TOTP string
}

type GenerateTOTP struct {
	Guard  GenerateTOTPGuard
	UserID string
}

func (cmd GenerateTOTP) Execute(ctx context.Context, bus command.Bus) (generateTOTPResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(generateTOTPResponse), errors.Tracef(err)
}

func (cmd GenerateTOTP) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd GenerateTOTP) request(ctx context.Context) (generateTOTPRequest, error) {
	var req generateTOTPRequest
	if !cmd.Guard.CanGenerateTOTP(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type GenerateTOTPHandler func(ctx context.Context, cmd GenerateTOTP) (generateTOTPResponse, error)

func NewGenerateTOTPHandler(broker event.Broker, users UserRepo) GenerateTOTPHandler {
	return func(ctx context.Context, cmd GenerateTOTP) (generateTOTPResponse, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return generateTOTPResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return generateTOTPResponse{}, errors.Tracef(err)
		}

		totp, err := user.GenerateTOTP()
		if err != nil {
			return generateTOTPResponse{}, errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return generateTOTPResponse{}, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		res := generateTOTPResponse{TOTP: totp}

		return res, nil
	}
}
