package system

import (
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const resendAPIKeyLength = 36

var (
	invalidResendAPIKeyChars = regexp.MustCompile(`[^0-9a-zA-Z_]`)
	validResendAPIKeySeq     = regexp.MustCompile(`^re_[0-9a-zA-Z_]+$`)
)

type ResendAPIKey string

func NewResendAPIKey(apiKey string) (ResendAPIKey, error) {
	if apiKey == "" {
		return "", nil
	}

	if rc := utf8.RuneCountInString(apiKey); rc != resendAPIKeyLength {
		return "", i18n.M("resend_api_key.error.incorrect_length", "required_length", resendAPIKeyLength)
	}

	if matches := invalidResendAPIKeyChars.FindAllString(apiKey, -1); len(matches) != 0 {
		return "", i18n.M("resend_api_key.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validResendAPIKeySeq.MatchString(apiKey) {
		return "", i18n.M("resend_api_key.error.invalid")
	}

	return ResendAPIKey(apiKey), nil
}

func (e ResendAPIKey) String() string {
	return string(e)
}
