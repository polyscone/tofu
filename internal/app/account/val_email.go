package account

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

var validEmail = errsx.Must(regexp.Compile("" +
	`^` +
	// Local part
	`[\w+-](\.?[\w+-]|[\w+-]){0,60}` +
	// Separator
	`@` +
	// Domain
	`[0-9A-Za-z](-?[0-9A-Za-z]|[0-9A-Za-z]){0,60}` +
	`\.[A-Za-z]{2,6}(\.[A-Za-z]{2,6})?` +
	`$`,
))

type Email string

func NewEmail(email string) (Email, error) {
	if strings.TrimSpace(email) == "" {
		return "", errors.New("cannot be empty")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		email = strings.TrimSpace(email)
		msg := strings.TrimPrefix(strings.ToLower(err.Error()), "mail: ")

		switch {
		case strings.Contains(msg, "missing '@'"):
			return "", errors.New("missing @ sign")

		case strings.HasPrefix(email, "@"):
			return "", errors.New("missing part before @ sign")

		case strings.HasSuffix(email, "@"):
			return "", errors.New("missing part after @ sign")
		}

		return "", errors.New(msg)
	}

	if addr.Name != "" {
		return "", errors.New("should not include a name")
	}

	if !validEmail.MatchString(addr.Address) {
		_, end, _ := strings.Cut(addr.Address, "@")
		if !strings.Contains(end, ".") {
			return "", errors.New("missing top-level domain")
		}

		return "", errors.New("contains invalid characters")
	}

	return Email(addr.Address), nil
}

func (e Email) String() string {
	return string(e)
}
