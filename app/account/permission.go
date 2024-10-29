package account

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const (
	permissionMinLength = 1
	permissionMaxLength = 50
)

var (
	invalidPermissionChars = regexp.MustCompile(`[^a-z0-9._]`)
	validPermissionSeq     = regexp.MustCompile(`^[a-z0-9._]+$`)
)

type Permission string

func NewPermission(name string) (Permission, error) {
	if strings.TrimSpace(name) == "" {
		return "", i18n.M("account.permission.error.empty")
	}

	if strings.ContainsAny(name, " \t\r\n") {
		return "", i18n.M("account.permission.error.has_whitespace")
	}
	if strings.ContainsAny(name, `"'`) {
		return "", i18n.M("account.permission.error.has_quotes")
	}

	rc := utf8.RuneCountInString(name)
	if rc < permissionMinLength {
		return "", i18n.M("account.permission.error.too_short", "min_length", permissionMinLength)
	}
	if rc > permissionMaxLength {
		return "", i18n.M("account.permission.error.too_long", "max_length", permissionMaxLength)
	}

	if matches := invalidPermissionChars.FindAllString(name, -1); len(matches) != 0 {
		return "", i18n.M("account.permission.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validPermissionSeq.MatchString(name) {
		return "", i18n.M("account.permission.error.invalid")
	}

	return Permission(name), nil
}

func (n Permission) String() string {
	return string(n)
}
