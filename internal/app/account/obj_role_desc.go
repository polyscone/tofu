package account

import (
	"math/rand"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	roleDescMaxLength    = 100
	validRoleDescPattern = `^[[:print:]\r\n]*$`
)

var (
	validRoleDesc     = errors.Must(regexp.Compile(validRoleDescPattern))
	roleDescGenerator = errors.Must(gen.NewPatternGenerator(validRoleDescPattern))
)

type RoleDesc string

func GenerateRoleDesc() RoleDesc {
	return RoleDesc(roleDescGenerator.Generate())
}

func NewRoleDesc(desc string) (RoleDesc, error) {
	rc := utf8.RuneCountInString(desc)
	if rc > roleDescMaxLength {
		return "", errors.Tracef("cannot be a over %v characters in length", roleDescMaxLength)
	}

	if !validRoleDesc.MatchString(desc) {
		return "", errors.Tracef("contains invalid characters")
	}

	return RoleDesc(desc), nil
}

func (d RoleDesc) String() string {
	return string(d)
}

func (d RoleDesc) Equal(rhs RoleDesc) bool {
	return d == rhs
}

func (d RoleDesc) Generate(rand *rand.Rand) any {
	return GenerateRoleDesc()
}

func (d RoleDesc) Invalidate(rand *rand.Rand, value any) any {
	return RoleDesc(errors.Must(gen.Pattern(`([^[:print:]\r\n]{1,100}|a{101,})`)))
}
