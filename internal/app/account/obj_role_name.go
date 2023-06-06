package account

import (
	"math/rand"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	roleNameMinLength    = 1
	roleNameMaxLength    = 30
	validRoleNamePattern = `^[ a-zA-Z0-9!#&()*+,./:_\-\\]{1,30}$`
)

var (
	validRoleName     = errors.Must(regexp.Compile(validRoleNamePattern))
	roleNameGenerator = errors.Must(gen.NewPatternGenerator(validRoleNamePattern))
)

type RoleName string

func GenerateRoleName() RoleName {
	for {
		name, err := NewRoleName(roleNameGenerator.GenerateLimit(roleNameMaxLength))
		if err == nil {
			return name
		}
	}
}

func NewRoleName(name string) (RoleName, error) {
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
	if rc < roleNameMinLength {
		return "", errors.Tracef("must be at least %v characters", roleNameMinLength)
	}
	if rc > roleNameMaxLength {
		return "", errors.Tracef("cannot be a over %v characters in length", roleNameMaxLength)
	}

	if !validRoleName.MatchString(name) {
		return "", errors.Tracef("contains invalid characters")
	}

	return RoleName(name), nil
}

func (n RoleName) String() string {
	return string(n)
}

func (n RoleName) Equal(rhs RoleName) bool {
	return n == rhs
}

func (n RoleName) Generate(rand *rand.Rand) any {
	return GenerateRoleName()
}

func (n RoleName) Invalidate(rand *rand.Rand, value any) any {
	return RoleName(errors.Must(gen.Pattern(`(|[^ a-zA-Z0-9!#&()*+,./:_\-\\]{1,30}|a{31,})`)))
}
