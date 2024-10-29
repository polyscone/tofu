package system

import (
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
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
		return "", i18n.M("twilio_sid.error.incorrect_length", "required_length", twilioSIDLength)
	}

	if matches := invalidTwilioSIDChars.FindAllString(sid, -1); len(matches) != 0 {
		return "", i18n.M("twilio_sid.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validTwilioSIDSeq.MatchString(sid) {
		return "", i18n.M("twilio_sid.error.invalid")
	}

	return TwilioSID(sid), nil
}

func (e TwilioSID) String() string {
	return string(e)
}
