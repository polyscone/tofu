package text

import (
	"fmt"
	"math/rand"
	"net/mail"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const supportedEmailPattern = "" +
	// Local part
	`[\w+-](\.?[\w+-]|[\w+-]){0,60}` +
	// Separator
	`@` +
	// Domain
	`[0-9A-Za-z](-?[0-9A-Za-z]|[0-9A-Za-z]){0,60}` +
	`\.[A-Za-z]{2,6}(\.[A-Za-z]{2,6})?`

var (
	supportedEmail = errors.Must(regexp.Compile(supportedEmailPattern))
	emailGenerator = errors.Must(gen.NewPatternGenerator(supportedEmailPattern))
)

type Email string

func GenerateEmail() Email {
	return Email(emailGenerator.Generate())
}

func NewEmail(email string) (Email, error) {
	if strings.TrimSpace(email) == "" {
		return "", errors.Tracef("cannot be empty")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		email = strings.TrimSpace(email)
		msg := strings.TrimPrefix(strings.ToLower(err.Error()), "mail: ")

		switch {
		case strings.Contains(msg, "missing '@'"):
			return "", errors.Tracef("missing @ sign")

		case strings.HasPrefix(email, "@"):
			return "", errors.Tracef("missing part before @ sign")

		case strings.HasSuffix(email, "@"):
			return "", errors.Tracef("missing part after @ sign")
		}

		return "", errors.Tracef(msg)
	}

	if addr.Name != "" {
		return "", errors.Tracef("should not include a name")
	}

	if !supportedEmail.MatchString(addr.Address) {
		return "", errors.Tracef("contains unsupported characters")
	}

	return Email(addr.Address), nil
}

func (e Email) String() string {
	return string(e)
}

func (e Email) Generate(rand *rand.Rand) any {
	return GenerateEmail()
}

func (e Email) Invalidate(rand *rand.Rand, value any) any {
	valid := value.(Email).String()

	var invalid string
	switch rand.Intn(7) {
	case 0:
		// Remove @ sign
		invalid = strings.ReplaceAll(valid, "@", "")

	case 1:
		// Mulitple @ signs
		ats := errors.Must(gen.Pattern("@{2,}"))
		invalid = strings.ReplaceAll(valid, "@", ats)

	case 2:
		// Add a name
		name := errors.Must(gen.Pattern(`[A-Z][0-9A-Za-z ]*[a-z]`))
		invalid = fmt.Sprintf("%v <%v>", name, valid)

	case 3:
		// Special characters without quotation
		special := errors.Must(gen.Pattern(`[(),:;<>\[\]\\]+`))
		invalid = strings.ReplaceAll(valid, "@", special+"@")

	case 4:
		// No TLD
		local, _, _ := strings.Cut(valid, "@")
		domain := errors.Must(gen.Pattern(`[a-z]+`))
		invalid = fmt.Sprintf("%v@%v", local, domain)

	case 5:
		// Space quoted local part
		_, domain, _ := strings.Cut(valid, "@")
		invalid = fmt.Sprintf(`" "@%v`, domain)

	case 6:
		// IPv4 domain
		local, _, _ := strings.Cut(valid, "@")
		ip := errors.Must(gen.Pattern(`[1-9]{3}\.[1-9]{3}\.[1-9]{3}\.[1-9]{3}`))
		invalid = fmt.Sprintf(`%v@[%v]`, local, ip)
	}

	return Email(invalid)
}
