package event

type PasswordResetRequested struct {
	Email string
}

type SignInMagicLinkRequested struct {
	Email string
}

type TOTPSMSRequested struct {
	Email string
	Tel   string
}
