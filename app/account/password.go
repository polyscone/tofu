package account

import (
	"crypto/subtle"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const (
	passwordMinLength = 8
	passwordMaxLength = 1000
)

type Password struct {
	_ [0]func() // Disallow comparison

	data []byte
}

func NewPassword(password string) (zero Password, _ error) {
	if strings.TrimSpace(password) == "" {
		return zero, i18n.M("account.password.error.empty")
	}

	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return zero, i18n.M("account.password.error.too_short", "min_length", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return zero, i18n.M("account.password.error.too_long", "max_length", passwordMaxLength)
	}

	return Password{data: []byte(password)}, nil
}

func (p Password) String() string {
	return string(p.data)
}

func (p Password) Equal(rhs Password) bool {
	return subtle.ConstantTimeCompare(p.data, rhs.data) == 1
}
