package text

import (
	"math/rand"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const supportedTelephonePattern = `\+\d(\d| )+`

var (
	supportedTelephone = errors.Must(regexp.Compile(supportedTelephonePattern))
	telephoneGenerator = errors.Must(gen.NewPatternGenerator(supportedTelephonePattern))
)

type Telephone string

func GenerateTelephone() Telephone {
	return Telephone(telephoneGenerator.Generate())
}

func NewTelephone(telephone string) (Telephone, error) {
	if strings.TrimSpace(telephone) == "" {
		return "", errors.Tracef("cannot be empty")
	}

	if !supportedTelephone.MatchString(telephone) {
		return "", errors.Tracef("not a valid telephone number")
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
