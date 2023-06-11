package account

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const (
	roleDescMaxLength    = 100
	validRoleDescPattern = `^[[:print:]\r\n]*$`
)

var (
	validRoleDesc     = errsx.Must(regexp.Compile(validRoleDescPattern))
	roleDescGenerator = errsx.Must(gen.NewPatternGenerator(validRoleDescPattern))
)

type RoleDesc string

func GenerateRoleDesc() RoleDesc {
	return RoleDesc(roleDescGenerator.Generate())
}

func NewRoleDesc(desc string) (RoleDesc, error) {
	rc := utf8.RuneCountInString(desc)
	if rc > roleDescMaxLength {
		return "", fmt.Errorf("cannot be a over %v characters in length", roleDescMaxLength)
	}

	if !validRoleDesc.MatchString(desc) {
		return "", errors.New("contains invalid characters")
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
	return RoleDesc(errsx.Must(gen.Pattern(`([^[:print:]\r\n]{1,100}|a{101,})`)))
}
