package system

import (
	"regexp"

	"github.com/polyscone/tofu/internal/i18n"
)

var (
	invalidTwilioTelChars = regexp.MustCompile(`[^\d+ ]`)
	validTwilioTelSeq     = regexp.MustCompile(`^\+\d(\d| )+$`)
)

type TwilioTel string

func NewTwilioTel(tel string) (TwilioTel, error) {
	if tel == "" {
		return "", nil
	}

	if matches := invalidTwilioTelChars.FindAllString(tel, -1); len(matches) != 0 {
		return "", i18n.M("twilio_tel.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validTwilioTelSeq.MatchString(tel) {
		return "", i18n.M("twilio_tel.error.invalid")
	}

	return TwilioTel(tel), nil
}

func (t TwilioTel) String() string {
	return string(t)
}
