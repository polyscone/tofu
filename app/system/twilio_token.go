package system

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/pkg/human"
)

const twilioTokenLength = 32

var (
	invalidTwilioTokenChars = regexp.MustCompile(`[^0-9a-f]`)
	validTwilioTokenSeq     = regexp.MustCompile(`^[0-9a-f]+$`)
)

type TwilioToken string

func NewTwilioToken(token string) (TwilioToken, error) {
	if token == "" {
		return "", nil
	}

	if rc := utf8.RuneCountInString(token); rc != twilioTokenLength {
		return "", fmt.Errorf("must be %v characters in length", twilioTokenLength)
	}

	if matches := invalidTwilioTokenChars.FindAllString(token, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validTwilioTokenSeq.MatchString(token) {
		return "", errors.New("must be exactly 32 hexadecimal characters")
	}

	return TwilioToken(token), nil
}

func (e TwilioToken) String() string {
	return string(e)
}
