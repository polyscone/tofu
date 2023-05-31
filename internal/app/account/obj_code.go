package account

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	mrand "math/rand"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validCodePattern = `^[A-Z2-7]+$`

var (
	validCode     = errors.Must(regexp.Compile(validCodePattern))
	codeGenerator = errors.Must(gen.NewPatternGenerator(validCodePattern))
)

type Code string

func GenerateCode() (Code, error) {
	code := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, code); err != nil {
		return "", errors.Tracef(err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(code)

	return Code(encoded), nil
}

func NewCode(code string) (Code, error) {
	if !validCode.MatchString(code) {
		return "", errors.Tracef("contains invalid characters")
	}

	return Code(code), nil
}

func (c Code) String() string {
	return string(c)
}

func (c Code) Generate(rand *mrand.Rand) any {
	return Code(codeGenerator.Generate())
}

func (c Code) Invalidate(rand *mrand.Rand, value any) any {
	return Code(errors.Must(gen.Pattern(`[^A-Z2-7]*`)))
}
