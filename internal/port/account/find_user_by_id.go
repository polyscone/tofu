package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type findUserByIDRequest struct {
	userID uuid.V4
}

type FindUserByID struct {
	UserID string
}

func (cmd FindUserByID) Execute(ctx context.Context, bus command.Bus) (findUserResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(findUserResponse), errors.Tracef(err)
}

func (cmd FindUserByID) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd FindUserByID) request() (findUserByIDRequest, error) {
	var req findUserByIDRequest
	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type FindUserByIDHandler func(ctx context.Context, cmd FindUserByID) (findUserResponse, error)

func NewFindUserByIDHandler(broker event.Broker, users UserRepo) FindUserByIDHandler {
	return func(ctx context.Context, cmd FindUserByID) (findUserResponse, error) {
		req, err := cmd.request()
		if err != nil {
			return findUserResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return findUserResponse{}, errors.Tracef(err)
		}

		return newFindUserResponse(user), nil
	}
}
