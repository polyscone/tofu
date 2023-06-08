package passport

type Passport struct {
	userID      int
	isSignedIn  bool
	permissions []string
}

type User struct {
	ID          int
	IsSignedIn  bool
	Permissions []string
}

func New(user User) Passport {
	return Passport{
		userID:      user.ID,
		isSignedIn:  user.IsSignedIn,
		permissions: user.Permissions,
	}
}

func (p Passport) UserID() int {
	return p.userID
}

func (p Passport) IsSignedIn() bool {
	return p.isSignedIn
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

func (p Passport) CanChangeTOTPTelephone(userID int) bool {
	return p.userID == userID
}

func (p Passport) CanChangeRoles(userID int) bool {
	return p.can(changeRoles)
}

func (p Passport) CanViewRoles() bool {
	return p.can(viewRoles)
}

func (p Passport) CanCreateRoles() bool {
	return p.can(createRoles)
}

func (p Passport) CanEditRoles() bool {
	return p.can(editRoles)
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
