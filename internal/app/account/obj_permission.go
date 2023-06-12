package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

const (
	permissionMinLength = 1
	permissionMaxLength = 50
)

var validPermission = errsx.Must(regexp.Compile(`^[a-z0-9:_]{1,50}$`))

type Permission string

func NewPermission(name string) (Permission, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("cannot be empty")
	}

	if strings.ContainsAny(name, "\n\r") {
		return "", errors.New("cannot contain line breaks")
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

	if !validPermission.MatchString(name) {
		return "", errors.New("contains invalid characters")
	}

	return Permission(name), nil
}

func (n Permission) String() string {
	return string(n)
}
