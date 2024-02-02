package guard

import "slices"

type User struct {
	ID          int
	IsSuper     bool
	Permissions []string
}

type Passport struct {
	userID      int
	isSuper     bool
	permissions []string

	Account Account
	System  System
}

func NewPassport(user User) Passport {
	p := Passport{
		userID:      user.ID,
		isSuper:     user.IsSuper,
		permissions: user.Permissions,
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
	return slices.Contains(p.permissions, query)
}
