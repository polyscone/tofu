package system

import (
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

const validTwilioTelPattern = `^\+\d(\d| )+$`

var validTwilioTel = errors.Must(regexp.Compile(validTwilioTelPattern))

type TwilioTel string

func NewTwilioTel(tel string) (TwilioTel, error) {
	if tel == "" {
		return "", nil
	}

	if !validTwilioTel.MatchString(tel) {
		return "", errors.Tracef("invalid phone number")
	}

	return TwilioTel(tel), nil
}

func (t TwilioTel) String() string {
	return string(t)
}
