package account

import "github.com/polyscone/tofu/internal/i18n"

const (
	TOTPMethodNone TOTPMethod = ""
	TOTPMethodApp  TOTPMethod = "app"
	TOTPMethodSMS  TOTPMethod = "sms"
)

type TOTPMethod string

func NewTOTPMethod(method string) (TOTPMethod, error) {
	switch TOTPMethod(method) {
	case TOTPMethodNone, TOTPMethodApp, TOTPMethodSMS:
		return TOTPMethod(method), nil
	}

	return "", i18n.M("account.totp_method.error.invalid", "invalid_method", method)
}

func (t TOTPMethod) String() string {
	return string(t)
}
