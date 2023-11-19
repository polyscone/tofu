package event

type PasswordResetRequested struct {
	Email string
}

type TOTPSMSRequested struct {
	Email string
	Tel   string
}
