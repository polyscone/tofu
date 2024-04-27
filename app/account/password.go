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

var validPasswordSeq = regexp.MustCompile(`^.+$`)

type Password struct {
	_ [0]func() // Disallow comparison

	data []byte
}

func NewPassword(password string) (zero Password, _ error) {
	if strings.TrimSpace(password) == "" {
		return zero, errors.New("cannot be empty")
	}

	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return zero, fmt.Errorf("must be at least %v characters in length", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return zero, fmt.Errorf("cannot be a over %v characters in length", passwordMaxLength)
	}

	if !validPasswordSeq.MatchString(password) {
		return zero, fmt.Errorf("must be between %v and %v characters in length", passwordMinLength, passwordMaxLength)
	}

	return Password{data: []byte(password)}, nil
}

func (p Password) String() string {
	return string(p.data)
}

func (p Password) Equal(rhs Password) bool {
	return subtle.ConstantTimeCompare(p.data, rhs.data) == 1
}
