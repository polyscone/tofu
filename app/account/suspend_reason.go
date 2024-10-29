package account

import (
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/i18n"
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
		return "", i18n.M("account.suspend_reason.error.too_long", "max_length", suspendedReasonMaxLength)
	}

	if matches := invalidSuspendedReasonChars.FindAllString(reason, -1); len(matches) != 0 {
		return "", i18n.M("account.suspend_reason.error.has_invalid_chars", "invalid_chars", matches)
	}

	if !validSuspendedReasonSeq.MatchString(reason) {
		return "", i18n.M("account.suspend_reason.error.invalid")
	}

	return SuspendedReason(reason), nil
}

func (s SuspendedReason) String() string {
	return string(s)
}
