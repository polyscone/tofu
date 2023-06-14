package system

import (
	"errors"
	"regexp"
)

var validTwilioToken = regexp.MustCompile(`^[0-9a-f]{32}$`)

type TwilioToken string

func NewTwilioToken(sid string) (TwilioToken, error) {
	if sid == "" {
		return "", nil
	}

	if !validTwilioToken.MatchString(sid) {
		return "", errors.New("must be exactly 32 hex characters")
	}

	return TwilioToken(sid), nil
}

func (e TwilioToken) String() string {
	return string(e)
}
