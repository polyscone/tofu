package text

import (
	"math/rand"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	optionalDescMinLength    = 0
	optionalDescMaxLength    = 500
	validOptionalDescPattern = `^[[:print:]\r\n]*$`
)

var (
	validOptionalDesc     = errors.Must(regexp.Compile(validOptionalDescPattern))
	optionalDescGenerator = errors.Must(gen.NewPatternGenerator(validOptionalDescPattern))
)

type OptionalDesc string

func GenerateOptionalDesc() OptionalDesc {
	return OptionalDesc(optionalDescGenerator.Generate())
}

func NewOptionalDesc(desc string) (OptionalDesc, error) {
	rc := utf8.RuneCountInString(desc)
	if rc < optionalDescMinLength {
		return "", errors.Tracef("must be at least %v characters", optionalDescMinLength)
	}
	if rc > optionalDescMaxLength {
		return "", errors.Tracef("cannot be a over %v characters in length", optionalDescMaxLength)
	}

	if !validOptionalDesc.MatchString(desc) {
		return "", errors.Tracef("contains invalid characters")
	}

	return OptionalDesc(desc), nil
}

func (d OptionalDesc) String() string {
	return string(d)
}

func (d OptionalDesc) Equal(rhs OptionalDesc) bool {
	return d == rhs
}

func (d OptionalDesc) Generate(rand *rand.Rand) any {
	optionalDescription, err := NewOptionalDesc(optionalDescGenerator.GenerateLimit(optionalDescMaxLength))
	for {
		if err == nil {
			return optionalDescription
		}

		optionalDescription, err = NewOptionalDesc(optionalDescGenerator.GenerateLimit(optionalDescMaxLength))
	}
}

func (d OptionalDesc) Invalidate(rand *rand.Rand, value any) any {
	return OptionalDesc(errors.Must(gen.Pattern(`([^[[:print:]]\r\n]{1,500}|a{501,})`)))
}
