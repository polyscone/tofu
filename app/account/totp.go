package account

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/human"
)

const totpLength = 6

var (
	invalidTOTPChars = regexp.MustCompile(`[^\d]`)
	validTOTPSeq     = regexp.MustCompile(`^\d+$`)
)

type TOTP string

func NewTOTP(totp string) (TOTP, error) {
	if rc := utf8.RuneCountInString(totp); rc != totpLength {
		return "", fmt.Errorf("must be %v characters in length", totpLength)
	}

	if matches := invalidTOTPChars.FindAllString(totp, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validTOTPSeq.MatchString(totp) {
		return "", fmt.Errorf("must be %v digits", totpLength)
	}

	return TOTP(totp), nil
}

func (t TOTP) String() string {
	return string(t)
}
