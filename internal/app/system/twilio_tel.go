package system

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/human"
)

var (
	invalidTwilioTelChars = regexp.MustCompile(`[^\d+ ]`)
	validTwilioTelSeq     = regexp.MustCompile(`^\+\d(\d| )+$`)
)

type TwilioTel string

func NewTwilioTel(tel string) (TwilioTel, error) {
	if tel == "" {
		return "", nil
	}

	if matches := invalidTwilioTelChars.FindAllString(tel, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validTwilioTelSeq.MatchString(tel) {
		return "", errors.New("must be in the format +12 3456 7890")
	}

	return TwilioTel(tel), nil
}

func (t TwilioTel) String() string {
	return string(t)
}
