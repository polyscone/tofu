package account

import "github.com/polyscone/tofu/internal/pkg/otp"

type TOTPKey []byte

func NewTOTPKey(algorithm otp.Algorithm) (TOTPKey, error) {
	key, err := otp.NewKey(nil, algorithm)
	if err != nil {
		return nil, err
	}

	return TOTPKey(key), nil
}
