package account

import (
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const totpLength = 6

var (
	invalidTOTPChars = regexp.MustCompile(`[^\d]`)
	validTOTPSeq     = regexp.MustCompile(`^\d+$`)
)

type TOTP string

func NewTOTP(totp string) (TOTP, error) {
	if rc := utf8.RuneCountInString(totp); rc != totpLength {
		return "", i18n.M("account.totp.error.incorrect_length", "required_length", totpLength)
	}

	if matches := invalidTOTPChars.FindAllString(totp, -1); len(matches) != 0 {
		return "", i18n.M("account.totp.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validTOTPSeq.MatchString(totp) {
		return "", i18n.M("account.totp.error.invalid", "required_length", totpLength)
	}

	return TOTP(totp), nil
}

func (t TOTP) String() string {
	return string(t)
}
