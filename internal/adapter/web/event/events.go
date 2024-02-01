package event

import "time"

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
