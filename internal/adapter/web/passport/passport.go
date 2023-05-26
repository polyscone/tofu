package passport

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/session"
)

type Passport struct {
	ctx      context.Context
	sessions *session.Manager
	user     *account.User
}

func New(ctx context.Context, sessions *session.Manager, user *account.User) Passport {
	return Passport{
		ctx:      ctx,
		sessions: sessions,
		user:     user,
	}
}

func (p Passport) UserID() int {
	return p.user.ID
}

func (p Passport) IsAuthenticated() bool {
	return p.sessions.GetBool(p.ctx, sess.IsAuthenticated)
}

func (p Passport) CanChangePassword(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) CanResetPassword(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) CanDisableTOTP(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) CanRegenerateRecoveryCodes(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) CanSetupTOTP(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) CanVerifyTOTP(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) CanChangeTOTPTelephone(userID int) bool {
	return p.user.ID == userID
}

func (p Passport) is(query string) bool {
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
			if query == permission.ID {
				return true
			}
		}
	}

	return false
}
