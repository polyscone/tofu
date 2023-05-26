package account

import (
	"math/rand"
	"regexp"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

const validTOTPPattern = `^\d{6}$`

var (
	validTOTP     = errors.Must(regexp.Compile(validTOTPPattern))
	totpGenerator = errors.Must(gen.NewPatternGenerator(validTOTPPattern))
)

type TOTP string

func NewTOTP(totp string) (TOTP, error) {
	if !validTOTP.MatchString(totp) {
		return "", errors.Tracef("must be 6 digits")
	}

	return TOTP(totp), nil
}

func (t TOTP) String() string {
	return string(t)
}

// Generate will return a well-formed TOTP.
// Since it does not use a key to actually generate a real TOTP this generator
// only produces valid TOTPs in the sense that they are not malformed.
func (t TOTP) Generate(rand *rand.Rand) any {
	return TOTP(totpGenerator.Generate())
}

func (t TOTP) Invalidate(rand *rand.Rand, value any) any {
	return TOTP(errors.Must(gen.Pattern(`([^\d]{6}|\w{6}|\d{0,5}|\d{7,}|.{0,5}|.{7,})`)))
}
