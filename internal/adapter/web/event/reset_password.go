package event

type ResetPasswordRequested struct {
	Email string
	Token string
}
