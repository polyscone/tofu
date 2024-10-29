package account

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const (
	roleNameMinLength = 1
	roleNameMaxLength = 30
)

var (
	invalidRoleNameChars = regexp.MustCompile(`[^[:print:]]`)
	validRoleNameSeq     = regexp.MustCompile(`^[[:print:]]+$`)
)

type RoleName string

func NewRoleName(name string) (RoleName, error) {
	if strings.TrimSpace(name) == "" {
		return "", i18n.M("account.role_name.error.empty")
	}

	rc := utf8.RuneCountInString(name)
	if rc < roleNameMinLength {
		return "", i18n.M("account.role_name.error.too_short", "min_length", roleNameMinLength)
	}
	if rc > roleNameMaxLength {
		return "", i18n.M("account.role_name.error.too_long", "max_length", roleNameMaxLength)
	}

	if matches := invalidRoleNameChars.FindAllString(name, -1); len(matches) != 0 {
		return "", i18n.M("account.role_name.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validRoleNameSeq.MatchString(name) {
		return "", i18n.M("account.role_name.error.invalid")
	}

	return RoleName(name), nil
}

func (n RoleName) String() string {
	return string(n)
}
