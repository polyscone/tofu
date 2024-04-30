package account

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/human"
)

const suspendedReasonMaxLength = 100

var (
	invalidSuspendedReasonChars = regexp.MustCompile(`[^[:print:]]`)
	validSuspendedReasonSeq     = regexp.MustCompile(`^[[:print:]]*$`)
)

type SuspendedReason string

func NewSuspendedReason(reason string) (SuspendedReason, error) {
	if reason == "" {
		return "", nil
	}

	rc := utf8.RuneCountInString(reason)
	if rc > suspendedReasonMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", suspendedReasonMaxLength)
	}

	if matches := invalidSuspendedReasonChars.FindAllString(reason, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validSuspendedReasonSeq.MatchString(reason) {
		return "", errors.New("can only contain latin characters")
	}

	return SuspendedReason(reason), nil
}

func (s SuspendedReason) String() string {
	return string(s)
}
