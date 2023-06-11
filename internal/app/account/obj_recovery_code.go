package account

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validCodePattern = `^[A-Z2-7]+$`

var (
	validCode     = errsx.Must(regexp.Compile(validCodePattern))
	codeGenerator = errsx.Must(gen.NewPatternGenerator(validCodePattern))
)

type RecoveryCode string

func GenerateRecoveryCode() (RecoveryCode, error) {
	code := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, code); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(code)

	return RecoveryCode(encoded), nil
}

func NewRecoveryCode(code string) (RecoveryCode, error) {
	if !validCode.MatchString(code) {
		return "", errors.New("contains invalid characters")
	}

	return RecoveryCode(code), nil
}

func (c RecoveryCode) String() string {
	return string(c)
}

func (c RecoveryCode) Generate(rand *mrand.Rand) any {
	return RecoveryCode(codeGenerator.Generate())
}

func (c RecoveryCode) Invalidate(rand *mrand.Rand, value any) any {
	return RecoveryCode(errsx.Must(gen.Pattern(`[^A-Z2-7]*`)))
}
