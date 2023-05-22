package account

type Registered struct {
	Email string
}

type Activated struct {
	Email string
}

type AuthenticatedWithPassword struct {
	Email          string
	IsAwaitingTOTP bool
}

type AuthenticatedWithTOTP struct {
	Email string
}

type AuthenticatedWithRecoveryCode struct {
	Email string
}

type DisabledTOTP struct {
	Email string
}

type RecoveryCodesRegenerated struct {
	Email string
}

type TOTPTelephoneChanged struct {
	Email        string
	OldTelephone string
	NewTelephone string
}

type PasswordChanged struct {
	Email string
}

type PasswordReset struct {
	Email string
}
