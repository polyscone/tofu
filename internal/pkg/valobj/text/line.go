package text

import (
	"math/rand"
	"regexp"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validLinePattern = `^[^\n\r]+$`

var (
	validLine     = errors.Must(regexp.Compile(validLinePattern))
	lineGenerator = errors.Must(gen.NewPatternGenerator(validLinePattern))
)

type Line string

func GenerateLine() Line {
	return Line(lineGenerator.Generate())
}

func NewLine(text string) (Line, error) {
	if text == "" {
		return "", errors.Tracef("cannot be empty")
	}

	if strings.ContainsAny(text, "\n\r") {
		return "", errors.Tracef("cannot contain line breaks")
	}

	if !validLine.MatchString(text) {
		return "", errors.Tracef("contains invalid characters")
	}

	return Line(text), nil
}

func (l Line) String() string {
	return string(l)
}

func (l Line) Generate(rand *rand.Rand) any {
	return GenerateLine()
}

func (l Line) Invalidate(rand *rand.Rand, value any) any {
	return Line(errors.Must(gen.Pattern(`(.+\n+.+|.+\r+.+)*`)))
}
