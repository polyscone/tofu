package testutil

import "bytes"

type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

func (h *PasswordHasher) EncodedPasswordHash(password []byte) ([]byte, error) {
	return password, nil
}

func (h *PasswordHasher) VerifyPasswordHash(password, encodedHash []byte) (bool, bool, error) {
	return bytes.Equal(password, encodedHash), false, nil
}
