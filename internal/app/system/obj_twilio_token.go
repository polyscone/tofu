package system

import (
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

const validTwilioTokenPattern = `^[0-9a-f]{32}$`

var validTwilioToken = errors.Must(regexp.Compile(validTwilioTokenPattern))

type TwilioToken string

func NewTwilioToken(sid string) (TwilioToken, error) {
	if sid == "" {
		return "", nil
	}

	if !validTwilioToken.MatchString(sid) {
		return "", errors.Tracef("must be exactly 32 hex characters")
	}

	return TwilioToken(sid), nil
}

func (e TwilioToken) String() string {
	return string(e)
}
