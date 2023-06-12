package account

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

const (
	passwordMinLength = 8
	passwordMaxLength = 100
)

var validPassword = errsx.Must(regexp.Compile(`^[[:print:]]{8,100}$`))

type Password []byte

func NewPassword(password string) (Password, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("cannot be empty")
	}

	if strings.ContainsAny(password, "\n\r") {
		return nil, errors.New("cannot contain line breaks")
	}

	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return nil, fmt.Errorf("must be at least %v characters", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return nil, fmt.Errorf("cannot be a over %v characters in length", passwordMaxLength)
	}

	if !validPassword.MatchString(password) {
		return nil, errors.New("contains invalid characters")
	}

	return Password(password), nil
}

func (p Password) String() string {
	return string(p)
}

func (p Password) Equal(rhs Password) bool {
	return bytes.Equal(p, rhs)
}
