package account

type SignedUp struct {
	Email string
}

type Activated struct {
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

type DisabledTOTP struct {
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

type PasswordReset struct {
	Email string
}

type RolesChanged struct {
	Email string
}