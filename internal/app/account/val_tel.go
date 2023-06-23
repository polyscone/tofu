package account

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/human"
)

var (
	invalidTelChars = regexp.MustCompile(`[^\d+ ]`)
	validTelSeq     = regexp.MustCompile(`^\+\d(\d| )+$`)
)

type Tel string

func NewTel(tel string) (Tel, error) {
	if strings.TrimSpace(tel) == "" {
		return "", errors.New("cannot be empty")
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
