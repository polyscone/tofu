package account

import (
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const roleDescMaxLength = 100

var (
	invalidRoleDescChars = regexp.MustCompile(`[^[:print:]]`)
	validRoleDescSeq     = regexp.MustCompile(`^[[:print:]]*$`)
)

type RoleDesc string

func NewRoleDesc(desc string) (RoleDesc, error) {
	if desc == "" {
		return "", nil
	}

	rc := utf8.RuneCountInString(desc)
	if rc > roleDescMaxLength {
		return "", i18n.M("account.role_description.error.too_long", "max_length", roleDescMaxLength)
	}

	if matches := invalidRoleDescChars.FindAllString(desc, -1); len(matches) != 0 {
		return "", i18n.M("account.role_description.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validRoleDescSeq.MatchString(desc) {
		return "", i18n.M("account.role_description.error.invalid")
	}

	return RoleDesc(desc), nil
}

func (d RoleDesc) String() string {
	return string(d)
}
