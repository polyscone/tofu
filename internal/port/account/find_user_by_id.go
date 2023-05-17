package account

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type findUserByIDRequest struct {
	userID uuid.V4
}

type findUserResponse struct {
	ID            string
	Email         string
	TOTPUseSMS    bool
	TOTPTelephone string
	TOTPKey       []byte
	TOTPAlgorithm string
	TOTPDigits    int
	TOTPPeriod    time.Duration
	RecoveryCodes []string
	Claims        []string
	Roles         []string
	Permissions   []string
}

type FindUserByID struct {
	UserID string
}

func (cmd FindUserByID) Execute(ctx context.Context, bus command.Bus) (findUserResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(findUserResponse), errors.Tracef(err)
}

func (cmd FindUserByID) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd FindUserByID) request(ctx context.Context) (findUserByIDRequest, error) {
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
		req, err := cmd.request(ctx)
		if err != nil {
			return findUserResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return findUserResponse{}, errors.Tracef(err)
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

		recoveryCodes := make([]string, len(user.RecoveryCodes))
		for i, code := range user.RecoveryCodes {
			recoveryCodes[i] = code.String()
		}

		res := findUserResponse{
			ID:            user.ID.String(),
			Email:         user.Email.String(),
			TOTPUseSMS:    user.TOTPUseSMS,
			TOTPTelephone: user.TOTPTelephone.String(),
			TOTPKey:       user.TOTPKey,
			TOTPAlgorithm: user.TOTPAlgorithm,
			TOTPDigits:    user.TOTPDigits,
			TOTPPeriod:    user.TOTPPeriod,
			RecoveryCodes: recoveryCodes,
			Claims:        claims,
			Roles:         roles,
			Permissions:   permissions,
		}

		return res, nil
	}
}
