package account

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"regexp"
)

var validCode = regexp.MustCompile(`^[A-Z2-7]+$`)

type RecoveryCode string

func NewRecoveryCode(code string) (RecoveryCode, error) {
	if !validCode.MatchString(code) {
		return "", errors.New("contains invalid characters")
	}

	return RecoveryCode(code), nil
}

func NewRandomRecoveryCode() (RecoveryCode, error) {
	code := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, code); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(code)

	return NewRecoveryCode(encoded)
}

func (c RecoveryCode) String() string {
	return string(c)
}
