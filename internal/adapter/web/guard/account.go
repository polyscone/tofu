package guard

type Account struct {
	*Passport
}

func (a Account) CanChangePassword(userID int) bool {
	return a.userID == userID
}

func (a Account) CanChoosePassword(userID int) bool {
	return a.userID == userID
}

func (a Account) CanResetPassword(userID int) bool {
	return a.userID == userID
}

func (a Account) CanDisableTOTP(userID int) bool {
	return a.userID == userID
}

func (a Account) CanRegenerateRecoveryCodes(userID int) bool {
	return a.userID == userID
}

func (a Account) CanSetupTOTP(userID int) bool {
	return a.userID == userID
}

func (a Account) CanVerifyTOTP(userID int) bool {
	return a.userID == userID
}

func (a Account) CanActivateTOTP(userID int) bool {
	return a.userID == userID
}

func (a Account) CanChangeTOTPTel(userID int) bool {
	return a.userID == userID
}

func (a Account) CanChangeRoles(userID int) bool {
	return a.can(changeRoles)
}

func (a Account) CanAssignSuperRole(userID int) bool {
	return a.isSuper
}

func (a Account) CanViewRoles() bool {
	return a.can(viewRoles)
}

func (a Account) CanCreateRoles() bool {
	return a.can(createRoles)
}

func (a Account) CanUpdateRoles() bool {
	return a.can(updateRoles)
}

func (a Account) CanDeleteRoles() bool {
	return a.can(deleteRoles)
}

func (a Account) CanViewUsers() bool {
	return a.can(viewUsers)
}

func (a Account) CanEditUsers() bool {
	return a.can(editUsers)
}