package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port"
)

type findUserByEmailRequest struct {
	email text.Email
}

type FindUserByEmail struct {
	Email string
}

func (cmd FindUserByEmail) Execute(ctx context.Context, bus command.Bus) (findUserResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(findUserResponse), errors.Tracef(err)
}

func (cmd FindUserByEmail) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd FindUserByEmail) request() (findUserByEmailRequest, error) {
	var req findUserByEmailRequest
	var err error
	var errs errors.Map

	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type FindUserByEmailHandler func(ctx context.Context, cmd FindUserByEmail) (findUserResponse, error)

func NewFindUserByEmailHandler(broker event.Broker, users UserRepo) FindUserByEmailHandler {
	return func(ctx context.Context, cmd FindUserByEmail) (findUserResponse, error) {
		req, err := cmd.request()
		if err != nil {
			return findUserResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, req.email)
		if err != nil {
			return findUserResponse{}, errors.Tracef(err)
		}

		return newFindUserResponse(user), nil
	}
}
