package text

import (
	"bytes"
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
	validNamePattern = `^[A-Za-z0-9!#&()*+,./:_\-\\]{1,100}$`
)

var (
	validName     = errors.Must(regexp.Compile(validNamePattern))
	nameGenerator = errors.Must(gen.NewPatternGenerator(validNamePattern))
)

type Name []byte

func GenerateName() Name {
	return Name(nameGenerator.Generate())
}

func NewName(name string) (Name, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.Tracef("cannot be empty")
	}

	if strings.ContainsAny(name, "\n\r") {
		return nil, errors.Tracef("cannot contain line breaks")
	}
	if strings.ContainsAny(name, `"'`) {
		return nil, errors.Tracef("cannot contain quotes")
	}

	rc := utf8.RuneCountInString(name)
	if rc < nameMinLength {
		return nil, errors.Tracef("must be at least %v characters", nameMinLength)
	}
	if rc > nameMaxLength {
		return nil, errors.Tracef("cannot be a over %v characters in length", nameMaxLength)
	}

	if !validName.MatchString(name) {
		return nil, errors.Tracef("contains invalid characters")
	}

	return Name(name), nil
}

func (n Name) String() string {
	return string(n)
}

func (n Name) Equal(rhs Name) bool {
	return bytes.Equal(n, rhs)
}

func (n Name) Generate(rand *rand.Rand) any {
	return Name(nameGenerator.GenerateLimit(nameMaxLength))
}

func (n Name) Invalidate(rand *rand.Rand, value any) any {
	return Name(errors.Must(gen.Pattern(`(|[^A-Za-z0-9!#&()*+,./:_\-\\]{1,100}|a{101,})`)))
}
