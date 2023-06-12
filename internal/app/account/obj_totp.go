package account

import (
	"errors"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

var validTOTP = errsx.Must(regexp.Compile(`^\d{6}$`))

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
