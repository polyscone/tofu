package system

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/human"
)

const twilioSIDLength = 34

var (
	invalidTwilioSIDChars = regexp.MustCompile(`[^AC0-9a-f]`)
	validTwilioSIDSeq     = regexp.MustCompile(`^AC[0-9a-f]+$`)
)

type TwilioSID string

func NewTwilioSID(sid string) (TwilioSID, error) {
	if sid == "" {
		return "", nil
	}

	if rc := utf8.RuneCountInString(sid); rc != twilioSIDLength {
		return "", fmt.Errorf("must be %v characters in length", twilioSIDLength)
	}

	if matches := invalidTwilioSIDChars.FindAllString(sid, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validTwilioSIDSeq.MatchString(sid) {
		return "", errors.New("must begin with AC and be followed by 32 hexadecimal characters")
	}

	return TwilioSID(sid), nil
}

func (e TwilioSID) String() string {
	return string(e)
}
