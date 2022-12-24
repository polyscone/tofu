package passport

import "github.com/polyscone/tofu/internal/pkg/valobj/uuid"

var Empty Passport

type Passport struct {
	claims      []string
	roles       []string
	permissions []string
}

func New(claims, roles, permissions []string) Passport {
	return Passport{
		claims:      claims,
		roles:       roles,
		permissions: permissions,
	}
}

func (p Passport) CanChangePassword(userID uuid.V4) bool { return p.is(userID.String()) }
func (p Passport) CanResetPassword(userID uuid.V4) bool  { return p.is(userID.String()) }
func (p Passport) CanSetupTOTP(userID uuid.V4) bool      { return p.is(userID.String()) }
func (p Passport) CanVerifyTOTP(userID uuid.V4) bool     { return p.is(userID.String()) }

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
