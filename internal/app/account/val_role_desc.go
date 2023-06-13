package account

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

const roleDescMaxLength = 100

var validRoleDesc = errsx.Must(regexp.Compile(`^[[:print:]]*$`))

type RoleDesc string

func NewRoleDesc(desc string) (RoleDesc, error) {
	rc := utf8.RuneCountInString(desc)
	if rc > roleDescMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", roleDescMaxLength)
	}

	if !validRoleDesc.MatchString(desc) {
		return "", errors.New("contains invalid characters")
	}

	return RoleDesc(desc), nil
}

func (d RoleDesc) String() string {
	return string(d)
}
