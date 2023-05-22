package account

import (
	"bytes"
	"math/rand"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	passwordMinLength        = 8
	passwordMaxLength        = 100
	supportedPasswordPattern = `[ -~]{8,100}`
)

var (
	supportedPassword = errors.Must(regexp.Compile(supportedPasswordPattern))
	passwordGenerator = errors.Must(gen.NewPatternGenerator(supportedPasswordPattern))
)

type Password []byte

func NewPassword(password string) (Password, error) {
	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return nil, errors.Tracef("must be at least %v characters", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return nil, errors.Tracef("passwords over %v characters are not supported", passwordMaxLength)
	}

	if !supportedPassword.MatchString(password) {
		return nil, errors.Tracef("contains unsupported characters")
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
	return Password(passwordGenerator.GenerateLimit(100))
}

func (p Password) Invalidate(rand *rand.Rand, value any) any {
	return Password(errors.Must(gen.Pattern(`(a{0,7}|[^ -~]{8,100}|a{101,})`)))
}
