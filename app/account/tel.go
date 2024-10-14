package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/human"
)

const telMaxLength = 100

var (
	invalidTelChars = regexp.MustCompile(`[^\d+ ]`)
	validTelSeq     = regexp.MustCompile(`^\+\d(\d| )+$`)
)

type Tel string

func NewTel(tel string) (Tel, error) {
	tel = strings.TrimSpace(tel)
	if tel == "" {
		return "", errors.New("cannot be empty")
	}

	tel = strings.Join(strings.Fields(tel), " ")

	if rc := utf8.RuneCountInString(tel); rc > telMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", telMaxLength)
	}

	if strings.Contains(tel, "+") && tel[0] != '+' {
		return "", errors.New("+ sign must come at the beginning")
	}

	if matches := invalidTelChars.FindAllString(tel, -1); len(matches) != 0 {
		return "", fmt.Errorf("cannot contain: %v", human.OrList(matches))
	}

	if !validTelSeq.MatchString(tel) {
		return "", errors.New("must be in the format +12 3456 7890")
	}

	return Tel(tel), nil
}

func (t Tel) String() string {
	return string(t)
}
