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

type findUserByEmailResponse struct {
	ID    string
	Email string
}

type FindUserByEmail struct {
	Email string
}

func (cmd FindUserByEmail) Execute(ctx context.Context, bus command.Bus) (findUserByEmailResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(findUserByEmailResponse), errors.Tracef(err)
}

func (cmd FindUserByEmail) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd FindUserByEmail) request(ctx context.Context) (findUserByEmailRequest, error) {
	var req findUserByEmailRequest
	var err error
	var errs errors.Map

	if req.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type FindUserByEmailHandler func(ctx context.Context, cmd FindUserByEmail) (findUserByEmailResponse, error)

func NewFindUserByEmailHandler(broker event.Broker, users UserRepo) FindUserByEmailHandler {
	return func(ctx context.Context, cmd FindUserByEmail) (findUserByEmailResponse, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return findUserByEmailResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, req.email)
		if err != nil {
			return findUserByEmailResponse{}, errors.Tracef(err)
		}

		res := findUserByEmailResponse{
			ID:    user.ID.String(),
			Email: user.Email.String(),
		}

		return res, nil
	}
}
