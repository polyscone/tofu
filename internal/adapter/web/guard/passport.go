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
}

func NewPassport(requireConfigSetup bool, user User) Passport {
	return Passport{
		requireConfigSetup: requireConfigSetup,
		userID:             user.ID,
		isSuper:            user.IsSuper,
		permissions:        user.Permissions,
	}
}

func (p Passport) CanViewConfig() bool {
	return p.requireConfigSetup || p.can(viewConfig)
}

func (p Passport) CanUpdateConfig() bool {
	return p.requireConfigSetup || p.can(updateConfig)
}

func (p Passport) CanChangePassword(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanResetPassword(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanDisableTOTP(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanRegenerateRecoveryCodes(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanSetupTOTP(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanVerifyTOTP(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanActivateTOTP(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanChangeTOTPTel(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanChangeRoles(userID int) bool {
	return p.can(changeRoles)
}

func (p Passport) CanAssignSuperRole(userID int) bool {
	return p.isSuper
}

func (p Passport) CanViewRoles() bool {
	return p.can(viewRoles)
}

func (p Passport) CanCreateRoles() bool {
	return p.can(createRoles)
}

func (p Passport) CanUpdateRoles() bool {
	return p.can(updateRoles)
}

func (p Passport) CanDeleteRoles() bool {
	return p.can(deleteRoles)
}

func (p Passport) CanViewUsers() bool {
	return p.can(viewUsers)
}

func (p Passport) CanEditUsers() bool {
	return p.can(editUsers)
}

func (p Passport) can(query string) bool {
	for _, permission := range p.permissions {
		if query == permission {
			return true
		}
	}

	return false
}
