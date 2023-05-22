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

const supportedRecoveryCodePattern = `^[A-Z2-7]+$`

var (
	supportedRecoveryCode = errors.Must(regexp.Compile(supportedRecoveryCodePattern))
	recoveryCodeGenerator = errors.Must(gen.NewPatternGenerator(supportedRecoveryCodePattern))
)

type RecoveryCode string

func NewRecoveryCode(code string) (RecoveryCode, error) {
	if !supportedRecoveryCode.MatchString(code) {
		return "", errors.Tracef("contains unsupported characters")
	}

	return RecoveryCode(code), nil
}

func GenerateRecoveryCode() (RecoveryCode, error) {
	code := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, code); err != nil {
		return "", errors.Tracef(err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(code)

	return RecoveryCode(encoded), nil
}

func (t RecoveryCode) String() string {
	return string(t)
}

func (t RecoveryCode) Generate(rand *mrand.Rand) any {
	return RecoveryCode(recoveryCodeGenerator.Generate())
}

func (t RecoveryCode) Invalidate(rand *mrand.Rand, value any) any {
	return RecoveryCode(errors.Must(gen.Pattern(`[^A-Z2-7]*`)))
}
