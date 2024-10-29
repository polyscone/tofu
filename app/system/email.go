package system

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const emailMaxLength = 100

var (
	invalidEmailChars = regexp.MustCompile(`[^a-zA-Z0-9+.@_~\-]`)
	validEmailSeq     = regexp.MustCompile(`^[a-zA-Z0-9+._~\-]+@[a-zA-Z0-9+._~\-]+(\.[a-zA-Z0-9+._~\-]+)+$`)
)

type Email string

func NewEmail(email string) (Email, error) {
	if strings.TrimSpace(email) == "" {
		return "", i18n.M("system.email.error.empty")
	}

	if strings.ContainsAny(email, " \t\r\n") {
		return "", i18n.M("system.email.error.contains_whitespace")
	}

	if strings.ContainsAny(email, `"'`) {
		return "", i18n.M("system.email.error.contains_quotes")
	}

	if rc := utf8.RuneCountInString(email); rc > emailMaxLength {
		return "", i18n.M("system.email.error.too_long", "max_length", emailMaxLength)
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		email = strings.TrimSpace(email)
		msg := strings.TrimPrefix(strings.ToLower(err.Error()), "mail: ")

		switch {
		case strings.Contains(msg, "missing '@'"):
			return "", i18n.M("system.email.error.missing_at")

		case strings.HasPrefix(email, "@"):
			return "", i18n.M("system.email.error.missing_local_part")

		case strings.HasSuffix(email, "@"):
			return "", i18n.M("system.email.error.missing_domain")
		}

		return "", i18n.M("system.email.error.other", "error", msg)
	}

	if addr.Name != "" {
		return "", i18n.M("system.email.error.has_name")
	}

	if matches := invalidEmailChars.FindAllString(addr.Address, -1); len(matches) != 0 {
		return "", i18n.M("system.email.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validEmailSeq.MatchString(addr.Address) {
		_, end, _ := strings.Cut(addr.Address, "@")
		if !strings.Contains(end, ".") {
			return "", i18n.M("system.email.error.missing_tld")
		}

		return "", i18n.M("system.email.error.not_an_email")
	}

	return Email(addr.Address), nil
}

func (e Email) String() string {
	return string(e)
}
