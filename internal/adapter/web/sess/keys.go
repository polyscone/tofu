package sess

const (
	// Global session keys
	Flash          = "global.flash"
	FlashImportant = "global.flash_important"
	Redirect       = "global.redirect"
	SortTopID      = "global.sort_top_id"
	HighlightID    = "global.highlight_id"

	// Account session keys
	UserID                   = "account.user_id"
	Email                    = "account.email"
	TOTPMethod               = "account.totp_method"
	HasActivatedTOTP         = "account.has_verified_totp"
	IsAwaitingTOTP           = "account.is_awaiting_totp"
	IsSignedIn               = "account.is_signed_in"
	KnownPasswordBreachCount = "account.password_known_breach_count"
)
