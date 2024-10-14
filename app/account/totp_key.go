package account

import "github.com/polyscone/tofu/internal/otp"

type TOTPKey struct {
	_ [0]func() // Disallow comparison

	data []byte
}

func NewTOTPKey(algorithm otp.Algorithm) (zero TOTPKey, _ error) {
	key, err := otp.NewKey(nil, algorithm)
	if err != nil {
		return zero, err
	}

	return TOTPKey{data: key}, nil
}
