package text

import (
	"math/rand"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	nameMinLength    = 1
	nameMaxLength    = 100
	validNamePattern = `^[ a-zA-Z0-9!#&()*+,./:_\-\\]{1,100}$`
)

var (
	validName     = errors.Must(regexp.Compile(validNamePattern))
	nameGenerator = errors.Must(gen.NewPatternGenerator(validNamePattern))
)

type Name string

func GenerateName() Name {
	return Name(nameGenerator.Generate())
}

func NewName(name string) (Name, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.Tracef("cannot be empty")
	}

	if strings.ContainsAny(name, "\n\r") {
		return "", errors.Tracef("cannot contain line breaks")
	}
	if strings.ContainsAny(name, `"'`) {
		return "", errors.Tracef("cannot contain quotes")
	}

	rc := utf8.RuneCountInString(name)
	if rc < nameMinLength {
		return "", errors.Tracef("must be at least %v characters", nameMinLength)
	}
	if rc > nameMaxLength {
		return "", errors.Tracef("cannot be a over %v characters in length", nameMaxLength)
	}

	if !validName.MatchString(name) {
		return "", errors.Tracef("contains invalid characters")
	}

	return Name(name), nil
}

func (n Name) String() string {
	return string(n)
}

func (n Name) Equal(rhs Name) bool {
	return n == rhs
}

func (n Name) Generate(rand *rand.Rand) any {
	name, err := NewName(nameGenerator.GenerateLimit(nameMaxLength))
	for {
		if err == nil {
			return name
		}

		name, err = NewName(nameGenerator.GenerateLimit(nameMaxLength))
	}
}

func (n Name) Invalidate(rand *rand.Rand, value any) any {
	return Name(errors.Must(gen.Pattern(`(|[^ a-zA-Z0-9!#&()*+,./:_\-\\]{1,100}|a{101,})`)))
}
