package account

type Invited struct {
	Email string
}

type SignedUp struct {
	Email string
}

type AlreadySignedUp struct {
	Email string
}

type SignedUpWithGoogle struct {
	Email string
}

type Verified struct {
	Email string
}

type SignedInWithPassword struct {
	Email string
}

type SignedInWithTOTP struct {
	Email string
}

type SignedInWithRecoveryCode struct {
	Email string
}

type SignedInWithGoogle struct {
	Email string
}

type TOTPDisabled struct {
	Email string
}

type TOTPResetRequested struct {
	Email string
}

type TOTPResetRequestApproved struct {
	Email string
}

type TOTPResetRequestDenied struct {
	Email string
}

type TOTPReset struct {
	Email string
}

type RecoveryCodesRegenerated struct {
	Email string
}

type TOTPTelChanged struct {
	Email  string
	OldTel string
	NewTel string
}

type PasswordChanged struct {
	Email string
}

type PasswordChosen struct {
	Email string
}

type PasswordReset struct {
	Email string
}

type RolesChanged struct {
	Email string
}
