package sess

const (
	// Global session keys
	Flash    = "global.flash"
	Redirect = "global.redirect"

	// Account session keys
	UserID                   = "account.user_id"
	Email                    = "account.email"
	HasVerifiedTOTP          = "account.has_verified_totp"
	TOTPUseSMS               = "account.totp_use_sms"
	IsAwaitingTOTP           = "account.is_awaiting_totp"
	IsAuthenticated          = "account.is_authenticated"
	PasswordKnownBreachCount = "account.password_known_breach_count"
)
