package system

import (
	"errors"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

const validTwilioSIDPattern = `^AC[0-9a-f]{32}$`

var validTwilioSID = errsx.Must(regexp.Compile(validTwilioSIDPattern))

type TwilioSID string

func NewTwilioSID(sid string) (TwilioSID, error) {
	if sid == "" {
		return "", nil
	}

	if !validTwilioSID.MatchString(sid) {
		return "", errors.New("must begin with AC and be followed by 32 hex characters")
	}

	return TwilioSID(sid), nil
}

func (e TwilioSID) String() string {
	return string(e)
}