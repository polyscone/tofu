package account

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	passwordMinLength = 8
	passwordMaxLength = 1000
)

var validPassword = regexp.MustCompile(`^.{8,1000}$`)

type Password struct {
	_ [0]func() // Disallow comparison

	data []byte
}

func NewPassword(password string) (zero Password, _ error) {
	if strings.TrimSpace(password) == "" {
		return zero, errors.New("cannot be empty")
	}

	if strings.ContainsAny(password, "\n\r") {
		return zero, errors.New("cannot contain line breaks")
	}

	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return zero, fmt.Errorf("must be at least %v characters", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return zero, fmt.Errorf("cannot be a over %v characters in length", passwordMaxLength)
	}

	if !validPassword.MatchString(password) {
		return zero, errors.New("contains invalid characters")
	}

	return Password{data: []byte(password)}, nil
}

func (p Password) String() string {
	return string(p.data)
}

func (p Password) Equal(rhs Password) bool {
	return subtle.ConstantTimeCompare(p.data, rhs.data) == 1
}
