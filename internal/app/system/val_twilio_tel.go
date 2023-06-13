package system

import (
	"errors"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

const validTwilioTelPattern = `^\+\d(\d| )+$`

var validTwilioTel = errsx.Must(regexp.Compile(validTwilioTelPattern))

type TwilioTel string

func NewTwilioTel(tel string) (TwilioTel, error) {
	if tel == "" {
		return "", nil
	}

	if !validTwilioTel.MatchString(tel) {
		return "", errors.New("invalid phone number")
	}

	return TwilioTel(tel), nil
}

func (t TwilioTel) String() string {
	return string(t)
}
