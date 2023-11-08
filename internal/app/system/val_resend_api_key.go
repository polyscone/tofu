package system

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/human"
)

const resendAPIKeyLength = 36

var (
	invalidResendAPIKeyChars = regexp.MustCompile(`[^0-9a-zA-Z_]`)
	validResendAPIKeySeq     = regexp.MustCompile(`^re_[0-9a-zA-Z_]+$`)
)

type ResendAPIKey string

func NewResendAPIKey(apiKey string) (ResendAPIKey, error) {
	if apiKey == "" {
		return "", nil
	}

	if rc := utf8.RuneCountInString(apiKey); rc != resendAPIKeyLength {
		return "", fmt.Errorf("must be %v characters in length", resendAPIKeyLength)
	}

	if matches := invalidResendAPIKeyChars.FindAllString(apiKey, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validResendAPIKeySeq.MatchString(apiKey) {
		return "", errors.New("must begin with AC and be followed by 32 hexadecimal characters")
	}

	return ResendAPIKey(apiKey), nil
}

func (e ResendAPIKey) String() string {
	return string(e)
}
