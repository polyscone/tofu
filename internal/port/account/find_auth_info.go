package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type findAuthInfoRequest struct {
	userID uuid.V4
}

type findAuthInfoResponse struct {
	Claims      []string
	Roles       []string
	Permissions []string
}

type FindAuthInfo struct {
	UserID string
}

func (cmd FindAuthInfo) Execute(ctx context.Context, bus command.Bus) (findAuthInfoResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(findAuthInfoResponse), errors.Tracef(err)
}

func (cmd FindAuthInfo) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd FindAuthInfo) request(ctx context.Context) (findAuthInfoRequest, error) {
	var req findAuthInfoRequest
	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type FindAuthInfoHandler func(ctx context.Context, cmd FindAuthInfo) (findAuthInfoResponse, error)

func NewFindAuthInfoHandler(broker event.Broker, users UserRepo) FindAuthInfoHandler {
	return func(ctx context.Context, cmd FindAuthInfo) (findAuthInfoResponse, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return findAuthInfoResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return findAuthInfoResponse{}, errors.Tracef(err)
		}

		claims := []string{user.ID.String()}
		for _, claim := range user.Claims {
			claims = append(claims, claim.String())
		}

		var roles []string
		var permissions []string
		for _, role := range user.Roles {
			roles = append(roles, role.Name)

			for _, permission := range role.Permissions {
				permissions = append(permissions, permission.String())
			}
		}

		res := findAuthInfoResponse{
			Claims:      claims,
			Roles:       roles,
			Permissions: permissions,
		}

		return res, nil
	}
}
