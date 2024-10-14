package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/human"
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
		return "", errors.New("cannot be empty")
	}

	rc := utf8.RuneCountInString(name)
	if rc < roleNameMinLength {
		return "", fmt.Errorf("must be at least %v characters", roleNameMinLength)
	}
	if rc > roleNameMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", roleNameMaxLength)
	}

	if matches := invalidRoleNameChars.FindAllString(name, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validRoleNameSeq.MatchString(name) {
		return "", errors.New("can only contain latin characters")
	}

	return RoleName(name), nil
}

func (n RoleName) String() string {
	return string(n)
}
