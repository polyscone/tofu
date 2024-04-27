package account

import "fmt"

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

	return "", fmt.Errorf("invalid TOTP method %q", method)
}

func (t TOTPMethod) String() string {
	return string(t)
}
