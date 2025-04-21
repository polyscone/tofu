package event

import "time"

type UserImpersonationStarted struct {
	ImposterEmail string
	UserEmail     string
}

type UserImpersonationStopped struct {
	ImposterEmail string
	UserEmail     string
}

type PasswordResetRequested struct {
	Email string
}

type SignInMagicLinkRequested struct {
	Email string
	TTL   time.Duration
}

type TOTPSMSRequested struct {
	Email string
	Tel   string
}
