package account

import (
	"errors"
	"fmt"
	"math/rand"
	"net/mail"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validEmailPattern = "" +
	`^` +
	// Local part
	`[\w+-](\.?[\w+-]|[\w+-]){0,60}` +
	// Separator
	`@` +
	// Domain
	`[0-9A-Za-z](-?[0-9A-Za-z]|[0-9A-Za-z]){0,60}` +
	`\.[A-Za-z]{2,6}(\.[A-Za-z]{2,6})?` +
	`$`

var (
	validEmail     = errsx.Must(regexp.Compile(validEmailPattern))
	emailGenerator = errsx.Must(gen.NewPatternGenerator(validEmailPattern))
)

type Email string

func GenerateEmail() Email {
	return Email(emailGenerator.Generate())
}

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
		ats := errsx.Must(gen.Pattern("@{2,}"))
		invalid = strings.ReplaceAll(valid, "@", ats)

	case 2:
		// Add a name
		name := errsx.Must(gen.Pattern(`[A-Z][0-9A-Za-z ]*[a-z]`))
		invalid = fmt.Sprintf("%v <%v>", name, valid)

	case 3:
		// Special characters without quotation
		special := errsx.Must(gen.Pattern(`[(),:;<>\[\]\\]+`))
		invalid = strings.ReplaceAll(valid, "@", special+"@")

	case 4:
		// No TLD
		local, _, _ := strings.Cut(valid, "@")
		domain := errsx.Must(gen.Pattern(`[a-z]+`))
		invalid = fmt.Sprintf("%v@%v", local, domain)

	case 5:
		// Space quoted local part
		_, domain, _ := strings.Cut(valid, "@")
		invalid = fmt.Sprintf(`" "@%v`, domain)

	case 6:
		// IPv4 domain
		local, _, _ := strings.Cut(valid, "@")
		ip := errsx.Must(gen.Pattern(`[1-9]{3}\.[1-9]{3}\.[1-9]{3}\.[1-9]{3}`))
		invalid = fmt.Sprintf(`%v@[%v]`, local, ip)
	}

	return Email(invalid)
}
