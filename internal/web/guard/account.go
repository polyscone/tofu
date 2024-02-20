package guard

type Account struct {
	*Passport
}

func (a Account) CanChangePassword(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanChoosePassword(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanResetPassword(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanDisableTOTP(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanResetTOTP(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanRegenerateRecoveryCodes(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanSetupTOTP(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanVerifyTOTP(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanActivateTOTP(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanChangeTOTPTel(userID string) bool {
	return a.UserID == userID
}

func (a Account) CanChangeRoles(userID string) bool {
	return a.can(changeRoles)
}

func (a Account) CanAssignSuperRole(userID string) bool {
	return a.IsSuper
}

func (a Account) CanSuspendUsers() bool {
	return a.can(suspendUsers)
}

func (a Account) CanUnsuspendUsers() bool {
	return a.can(unsuspendUsers)
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

func (a Account) CanInviteUsers() bool {
	return a.can(inviteUsers)
}

func (a Account) CanActivateUsers() bool {
	return a.can(activateUsers)
}

func (a Account) CanReviewTOTPResets() bool {
	return a.can(reviewTOTPResets)
}
