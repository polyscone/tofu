package account

import (
	"errors"
	"math/rand"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validTelPattern = `^\+\d(\d| )+$`

var (
	validTel     = errsx.Must(regexp.Compile(validTelPattern))
	telGenerator = errsx.Must(gen.NewPatternGenerator(validTelPattern))
)

type Tel string

func GenerateTel() Tel {
	return Tel(telGenerator.Generate())
}

func NewTel(tel string) (Tel, error) {
	if strings.TrimSpace(tel) == "" {
		return "", errors.New("cannot be empty")
	}

	if !validTel.MatchString(tel) {
		return "", errors.New("invalid phone number")
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
	return Tel(errsx.Must(gen.Pattern(`(\d|\s|\w)*`)))
}
