package domain

type TOTPKey []byte

func NewTOTPKey(key []byte) TOTPKey {
	if key == nil {
		key = make([]byte, 0)
	}

	return TOTPKey(key)
}
