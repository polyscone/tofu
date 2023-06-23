package account

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/human"
)

const recoveryCodeLength = 13

var (
	invalidRecoveryCodeChars = regexp.MustCompile(`[^A-Z2-7]`)
	validRecoveryCodeSeq     = regexp.MustCompile(`^[A-Z2-7]+$`)
)

type RecoveryCode string

func NewRecoveryCode(code string) (RecoveryCode, error) {
	if strings.TrimSpace(code) == "" {
		return "", errors.New("cannot be empty")
	}

	if strings.ContainsAny(code, " \t\n\r") {
		return "", errors.New("cannot contain whitespace")
	}
	if strings.ContainsAny(code, `"'`) {
		return "", errors.New("cannot contain quotes")
	}

	if rc := utf8.RuneCountInString(code); rc != recoveryCodeLength {
		return "", fmt.Errorf("must be %v characters in length", recoveryCodeLength)
	}

	if matches := invalidRecoveryCodeChars.FindAllString(code, -1); len(matches) != 0 {
		return "", fmt.Errorf("contains invalid characters: %v", human.List(matches))
	}

	if !validRecoveryCodeSeq.MatchString(code) {
		return "", errors.New("can only contain uppercase characters between A-Z and 2-7")
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

func (c RecoveryCode) EqualHash(rhs []byte) bool {
	sum := sha256.Sum256([]byte(c))
	hash := sum[:]

	return subtle.ConstantTimeCompare(hash, rhs) == 1
}
