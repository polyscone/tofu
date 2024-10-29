package account

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const telMaxLength = 100

var (
	invalidTelChars = regexp.MustCompile(`[^\d+ ]`)
	validTelSeq     = regexp.MustCompile(`^\+\d(\d| )+$`)
)

type Tel string

func NewTel(tel string) (Tel, error) {
	tel = strings.TrimSpace(tel)
	if tel == "" {
		return "", i18n.M("account.tel.error.empty")
	}

	tel = strings.Join(strings.Fields(tel), " ")

	if rc := utf8.RuneCountInString(tel); rc > telMaxLength {
		return "", i18n.M("account.tel.error.too_long", "max_length", telMaxLength)
	}

	if strings.Contains(tel, "+") && tel[0] != '+' {
		return "", i18n.M("account.tel.error.incorrect_plus_position")
	}

	if matches := invalidTelChars.FindAllString(tel, -1); len(matches) != 0 {
		return "", i18n.M("account.tel.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validTelSeq.MatchString(tel) {
		return "", i18n.M("account.tel.error.invalid")
	}

	return Tel(tel), nil
}

func (t Tel) String() string {
	return string(t)
}
