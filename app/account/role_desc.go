package account

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/human"
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
		return "", fmt.Errorf("cannot be a over %v characters in length", roleDescMaxLength)
	}

	if matches := invalidRoleDescChars.FindAllString(desc, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validRoleDescSeq.MatchString(desc) {
		return "", errors.New("can only contain latin characters")
	}

	return RoleDesc(desc), nil
}

func (d RoleDesc) String() string {
	return string(d)
}
