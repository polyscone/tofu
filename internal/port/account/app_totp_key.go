package account

import (
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
)

type TOTPKey []byte

func NewTOTPKey(algorithm otp.Alg) (TOTPKey, error) {
	key, err := otp.NewKey(nil, algorithm)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	return TOTPKey(key), nil
}
