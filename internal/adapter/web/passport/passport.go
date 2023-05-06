package passport

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

// Passport is a wrapper around a session manager focused on the session for a
// single given context.
// The focus on a single context means that it can also implement guard
// interfaces for use in port commands.
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

func (p Passport) Renew() error {
	return p.sessions.Renew(p.ctx)
}

func (p Passport) Set(key string, value any) {
	p.sessions.Set(p.ctx, key, value)
}

func (p Passport) Get(key string) any {
	return p.sessions.Get(p.ctx, key)
}

func (p Passport) Delete(key string) {
	p.sessions.Delete(p.ctx, key)
}

func (p Passport) Has(key string) bool {
	return p.sessions.Has(p.ctx, key)
}

func (p Passport) GetBool(key string) bool {
	return p.sessions.GetBool(p.ctx, key)
}

func (p Passport) PopBool(key string) bool {
	return p.sessions.PopBool(p.ctx, key)
}

func (p Passport) GetInt(key string) int {
	return p.sessions.GetInt(p.ctx, key)
}

func (p Passport) PopInt(key string) int {
	return p.sessions.PopInt(p.ctx, key)
}

func (p Passport) GetFloat32(key string) float32 {
	return p.sessions.GetFloat32(p.ctx, key)
}

func (p Passport) PopFloat32(key string) float32 {
	return p.sessions.PopFloat32(p.ctx, key)
}

func (p Passport) GetFloat64(key string) float64 {
	return p.sessions.GetFloat64(p.ctx, key)
}

func (p Passport) PopFloat64(key string) float64 {
	return p.sessions.PopFloat64(p.ctx, key)
}

func (p Passport) GetString(key string) string {
	return p.sessions.GetString(p.ctx, key)
}

func (p Passport) PopString(key string) string {
	return p.sessions.PopString(p.ctx, key)
}
