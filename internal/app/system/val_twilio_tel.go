package system

import (
	"errors"
	"regexp"
)

var validTwilioTel = regexp.MustCompile(`^\+\d(\d| )+$`)

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
