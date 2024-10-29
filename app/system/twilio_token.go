package system

import (
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const twilioTokenLength = 32

var (
	invalidTwilioTokenChars = regexp.MustCompile(`[^0-9a-f]`)
	validTwilioTokenSeq     = regexp.MustCompile(`^[0-9a-f]+$`)
)

type TwilioToken string

func NewTwilioToken(token string) (TwilioToken, error) {
	if token == "" {
		return "", nil
	}

	if rc := utf8.RuneCountInString(token); rc != twilioTokenLength {
		return "", i18n.M("twilio_token.error.incorrect_length", "required_length", twilioTokenLength)
	}

	if matches := invalidTwilioTokenChars.FindAllString(token, -1); len(matches) != 0 {
		return "", i18n.M("twilio_token.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validTwilioTokenSeq.MatchString(token) {
		return "", i18n.M("twilio_token.error.invalid")
	}

	return TwilioToken(token), nil
}

func (e TwilioToken) String() string {
	return string(e)
}
