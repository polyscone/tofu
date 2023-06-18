package guard

type User struct {
	ID          int
	IsSuper     bool
	Permissions []string
}

type Passport struct {
	requireConfigSetup bool
	userID             int
	isSuper            bool
	permissions        []string

	Account Account
	System  System
}

func NewPassport(requireConfigSetup bool, user User) Passport {
	p := Passport{
		requireConfigSetup: requireConfigSetup,
		userID:             user.ID,
		isSuper:            user.IsSuper,
		permissions:        user.Permissions,
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
	for _, permission := range p.permissions {
		if query == permission {
			return true
		}
	}

	return false
}
