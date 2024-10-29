package account

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
)

const recoveryCodeLength = 13

var (
	invalidRecoveryCodeChars = regexp.MustCompile(`[^A-Z2-7]`)
	validRecoveryCodeSeq     = regexp.MustCompile(`^[A-Z2-7]+$`)
)

type RecoveryCode string

func NewRecoveryCode(code string) (RecoveryCode, error) {
	if strings.TrimSpace(code) == "" {
		return "", i18n.M("account.recovery_code.error.empty")
	}

	if strings.ContainsAny(code, " \t\r\n") {
		return "", i18n.M("account.recovery_code.error.has_whitespace")
	}
	if strings.ContainsAny(code, `"'`) {
		return "", i18n.M("account.recovery_code.error.has_quotes")
	}

	if rc := utf8.RuneCountInString(code); rc != recoveryCodeLength {
		return "", i18n.M("account.recovery_code.error.incorrect_length", "required_length", recoveryCodeLength)
	}

	if matches := invalidRecoveryCodeChars.FindAllString(code, -1); len(matches) != 0 {
		return "", i18n.M("account.recovery_code.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validRecoveryCodeSeq.MatchString(code) {
		return "", i18n.M("account.recovery_code.error.invalid")
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
