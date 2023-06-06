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
	permissionMinLength    = 1
	permissionMaxLength    = 50
	validPermissionPattern = `^[a-z0-9:_]{1,50}$`
)

var (
	validPermission     = errors.Must(regexp.Compile(validPermissionPattern))
	permissionGenerator = errors.Must(gen.NewPatternGenerator(validPermissionPattern))
)

type Permission string

func GeneratePermission() Permission {
	return Permission(permissionGenerator.Generate())
}

func NewPermission(name string) (Permission, error) {
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
	if rc < permissionMinLength {
		return "", errors.Tracef("must be at least %v characters", permissionMinLength)
	}
	if rc > permissionMaxLength {
		return "", errors.Tracef("cannot be a over %v characters in length", permissionMaxLength)
	}

	if !validPermission.MatchString(name) {
		return "", errors.Tracef("contains invalid characters")
	}

	return Permission(name), nil
}

func (n Permission) String() string {
	return string(n)
}

func (n Permission) Equal(rhs Permission) bool {
	return n == rhs
}

func (n Permission) Generate(rand *rand.Rand) any {
	return GeneratePermission()
}

func (n Permission) Invalidate(rand *rand.Rand, value any) any {
	return Permission(errors.Must(gen.Pattern(`(|[^a-z0-9:_]{1,50}|a{51,})`)))
}
