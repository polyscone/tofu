package account

import (
	"math/rand"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validTelPattern = `^\+\d(\d| )+$`

var (
	validTel     = errors.Must(regexp.Compile(validTelPattern))
	telGenerator = errors.Must(gen.NewPatternGenerator(validTelPattern))
)

type Tel string

func GenerateTel() Tel {
	return Tel(telGenerator.Generate())
}

func NewTel(tel string) (Tel, error) {
	if strings.TrimSpace(tel) == "" {
		return "", errors.Tracef("cannot be empty")
	}

	if !validTel.MatchString(tel) {
		return "", errors.Tracef("invalid phone number")
	}

	return Tel(tel), nil
}

func (t Tel) String() string {
	return string(t)
}

func (t Tel) Generate(rand *rand.Rand) any {
	return GenerateTel()
}

func (t Tel) Invalidate(rand *rand.Rand, value any) any {
	return Tel(errors.Must(gen.Pattern(`(\d|\s|\w)*`)))
}
