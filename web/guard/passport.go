package guard

import (
	"slices"

	"github.com/polyscone/tofu/app/account"
)

type Passport struct {
	UserID      string
	IsSuper     bool
	Permissions []string

	Account Account
	System  System
}

func NewPassport(user *account.User, superRoleID string) Passport {
	var p Passport

	if user != nil {
		p.UserID = user.ID
		p.IsSuper = slices.ContainsFunc(user.Roles, func(role *account.Role) bool {
			return role.ID == superRoleID
		})
		p.Permissions = user.Permissions()
	}

	p.Account = Account{Passport: &p}
	p.System = System{Passport: &p}

	return p
}

func (p Passport) CanAccessAdmin() bool {
	switch {
	case p.Account.CanViewUsers(),
		p.Account.CanViewRoles(),
		p.System.CanViewConfig():

		return true

	default:
		return false
	}
}

func (p Passport) can(query string) bool {
	return slices.Contains(p.Permissions, query)
}
