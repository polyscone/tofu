package passport

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

type Passport struct {
	ctx         context.Context
	sessions    *session.Manager
	userID      string
	claims      []string
	roles       []string
	permissions []string
}

func New(ctx context.Context, sessions *session.Manager, userID string, claims, roles, permissions []string) Passport {
	return Passport{
		ctx:         ctx,
		sessions:    sessions,
		userID:      userID,
		claims:      claims,
		roles:       roles,
		permissions: permissions,
	}
}

func (p Passport) UserID() string {
	return p.userID
}

func (p Passport) IsAuthenticated() bool {
	return p.sessions.GetBool(p.ctx, sess.IsAuthenticated)
}

func (p Passport) CanChangePassword(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanResetPassword(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanDisableTOTP(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanRegenerateRecoveryCodes(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanSetupTOTP(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanVerifyTOTP(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanChangeTOTPTelephone(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) CanGenerateTOTP(userID uuid.V4) bool {
	return p.userID == userID.String()
}

func (p Passport) is(query string) bool {
	for _, claim := range p.claims {
		if query == claim {
			return true
		}
	}

	for _, role := range p.claims {
		if query == role {
			return true
		}
	}

	return false
}

func (p Passport) can(query string) bool {
	for _, permission := range p.permissions {
		if query == permission {
			return true
		}
	}

	return false
}
