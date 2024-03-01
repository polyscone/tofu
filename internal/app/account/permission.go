package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/human"
)

const (
	permissionMinLength = 1
	permissionMaxLength = 50
)

var (
	invalidPermissionChars = regexp.MustCompile(`[^a-z0-9:_]`)
	validPermissionSeq     = regexp.MustCompile(`^[a-z0-9:_]+$`)
)

type Permission string

func NewPermission(name string) (Permission, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("cannot be empty")
	}

	if strings.ContainsAny(name, " \t\r\n") {
		return "", errors.New("cannot contain whitespace")
	}
	if strings.ContainsAny(name, `"'`) {
		return "", errors.New("cannot contain quotes")
	}

	rc := utf8.RuneCountInString(name)
	if rc < permissionMinLength {
		return "", fmt.Errorf("must be at least %v characters", permissionMinLength)
	}
	if rc > permissionMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", permissionMaxLength)
	}

	if matches := invalidPermissionChars.FindAllString(name, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validPermissionSeq.MatchString(name) {
		return "", errors.New("can only contain letters, numbers, underscores, and colons, e.g. abc_123:def_456")
	}

	return Permission(name), nil
}

func (n Permission) String() string {
	return string(n)
}
