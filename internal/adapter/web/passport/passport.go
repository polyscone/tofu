package passport

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/query"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

type Passport struct {
	ctx      context.Context
	sessions *session.Manager
	user     query.AccountUser
}

func New(ctx context.Context, sessions *session.Manager, user query.AccountUser) Passport {
	return Passport{
		ctx:      ctx,
		sessions: sessions,
		user:     user,
	}
}

func (p Passport) UserID() string {
	return p.user.ID
}

func (p Passport) IsAuthenticated() bool {
	return p.sessions.GetBool(p.ctx, sess.IsAuthenticated)
}

func (p Passport) CanChangePassword(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanResetPassword(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanDisableTOTP(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanRegenerateRecoveryCodes(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanSetupTOTP(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanVerifyTOTP(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanChangeTOTPTelephone(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) CanGenerateTOTP(userID uuid.V4) bool {
	return p.user.ID == userID.String()
}

func (p Passport) is(query string) bool {
	for _, claim := range p.user.Claims {
		if query == claim {
			return true
		}
	}

	for _, role := range p.user.Roles {
		if query == role.Name {
			return true
		}
	}

	return false
}

func (p Passport) can(query string) bool {
	for _, role := range p.user.Roles {
		for _, permission := range role.Permissions {
			if query == permission {
				return true
			}
		}
	}

	return false
}
