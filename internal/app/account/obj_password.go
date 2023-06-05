package account

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
	passwordMinLength    = 8
	passwordMaxLength    = 100
	validPasswordPattern = `^[[:print:]]{8,100}$` // [[:print:]] ≡ [ -~]
)

var (
	validPassword     = errors.Must(regexp.Compile(validPasswordPattern))
	passwordGenerator = errors.Must(gen.NewPatternGenerator(validPasswordPattern))
)

type Password []byte

func GeneratePassword() Password {
	return Password(passwordGenerator.Generate())
}

func NewPassword(password string) (Password, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errors.Tracef("cannot be empty")
	}

	if strings.ContainsAny(password, "\n\r") {
		return nil, errors.Tracef("cannot contain line breaks")
	}

	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return nil, errors.Tracef("must be at least %v characters", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return nil, errors.Tracef("cannot be a over %v characters in length", passwordMaxLength)
	}

	if !validPassword.MatchString(password) {
		return nil, errors.Tracef("contains invalid characters")
	}

	return Password(password), nil
}

func (p Password) String() string {
	return string(p)
}

func (p Password) Equal(rhs Password) bool {
	return bytes.Equal(p, rhs)
}

func (p Password) Generate(rand *rand.Rand) any {
	return Password(passwordGenerator.GenerateLimit(passwordMaxLength))
}

func (p Password) Invalidate(rand *rand.Rand, value any) any {
	return Password(errors.Must(gen.Pattern(`(a{0,7}|[^ -~]{8,100}|a{101,})`)))
}
