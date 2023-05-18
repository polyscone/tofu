package account

import (
	"time"

	"github.com/polyscone/tofu/internal/port/account/domain"
)

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

func newFindUserResponse(user domain.User) findUserResponse {
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

	return findUserResponse{
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
}
