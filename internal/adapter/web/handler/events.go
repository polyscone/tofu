package handler

type ResetPasswordRequested struct {
	Email string
	Token string
}

type TOTPSMSRequested struct {
	Email string
}
