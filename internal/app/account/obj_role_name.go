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
	roleNameMinLength = 1
	roleNameMaxLength = 30
)

var validRoleName = errsx.Must(regexp.Compile(`^[ a-zA-Z0-9!#&()*+,./:_\-\\]{1,30}$`))

type RoleName string

func NewRoleName(name string) (RoleName, error) {
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
	if rc < roleNameMinLength {
		return "", fmt.Errorf("must be at least %v characters", roleNameMinLength)
	}
	if rc > roleNameMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", roleNameMaxLength)
	}

	if !validRoleName.MatchString(name) {
		return "", errors.New("contains invalid characters")
	}

	return RoleName(name), nil
}

func (n RoleName) String() string {
	return string(n)
}
