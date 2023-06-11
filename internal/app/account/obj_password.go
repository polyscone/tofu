package account

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	passwordMinLength    = 8
	passwordMaxLength    = 100
	validPasswordPattern = `^[[:print:]]{8,100}$` // [[:print:]] ≡ [ -~]
)

var (
	validPassword     = errsx.Must(regexp.Compile(validPasswordPattern))
	passwordGenerator = errsx.Must(gen.NewPatternGenerator(validPasswordPattern))
)

type Password []byte

func GeneratePassword() Password {
	return Password(passwordGenerator.Generate())
}

func NewPassword(password string) (Password, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("cannot be empty")
	}

	if strings.ContainsAny(password, "\n\r") {
		return nil, errors.New("cannot contain line breaks")
	}

	rc := utf8.RuneCountInString(password)
	if rc < passwordMinLength {
		return nil, fmt.Errorf("must be at least %v characters", passwordMinLength)
	}
	if rc > passwordMaxLength {
		return nil, fmt.Errorf("cannot be a over %v characters in length", passwordMaxLength)
	}

	if !validPassword.MatchString(password) {
		return nil, errors.New("contains invalid characters")
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
	return Password(errsx.Must(gen.Pattern(`(a{0,7}|[^ -~]{8,100}|a{101,})`)))
}
