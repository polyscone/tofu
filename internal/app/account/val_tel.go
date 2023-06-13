package account

import (
	"errors"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

var validTel = errsx.Must(regexp.Compile(`^\+\d(\d| )+$`))

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
