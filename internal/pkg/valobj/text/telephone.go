package text

import (
	"math/rand"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validTelephonePattern = `^\+\d(\d| )+$`

var (
	validTelephone     = errors.Must(regexp.Compile(validTelephonePattern))
	telephoneGenerator = errors.Must(gen.NewPatternGenerator(validTelephonePattern))
)

type Telephone string

func GenerateTelephone() Telephone {
	return Telephone(telephoneGenerator.Generate())
}

func NewTelephone(telephone string) (Telephone, error) {
	if strings.TrimSpace(telephone) == "" {
		return "", errors.Tracef("cannot be empty")
	}

	if !validTelephone.MatchString(telephone) {
		return "", errors.Tracef("invalid telephone number")
	}

	return Telephone(telephone), nil
}

func (t Telephone) String() string {
	return string(t)
}

func (t Telephone) Generate(rand *rand.Rand) any {
	return GenerateTelephone()
}

func (t Telephone) Invalidate(rand *rand.Rand, value any) any {
	return Telephone(errors.Must(gen.Pattern(`(\d|\s|\w)*`)))
}
