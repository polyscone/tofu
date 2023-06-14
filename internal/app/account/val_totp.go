package account

import (
	"errors"
	"regexp"
)

var validTOTP = regexp.MustCompile(`^\d{6}$`)

type TOTP string

func NewTOTP(totp string) (TOTP, error) {
	if !validTOTP.MatchString(totp) {
		return "", errors.New("must be 6 digits")
	}

	return TOTP(totp), nil
}

func (t TOTP) String() string {
	return string(t)
}
