package account

import (
	"errors"
	"regexp"
	"strings"
)

var validTel = regexp.MustCompile(`^\+\d(\d| )+$`)

type Tel string

func NewTel(tel string) (Tel, error) {
	if strings.TrimSpace(tel) == "" {
		return "", errors.New("cannot be empty")
	}

	if !validTel.MatchString(tel) {
		return "", errors.New("invalid phone number")
	}

	return Tel(tel), nil
}

func (t Tel) String() string {
	return string(t)
}
