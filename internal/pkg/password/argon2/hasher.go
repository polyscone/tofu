package argon2

import "github.com/polyscone/tofu/internal/pkg/size"

type Hasher struct {
	params Params
}

func NewHasher(params Params) *Hasher {
	return &Hasher{params: params}
}

func (h *Hasher) EncodedHash(password []byte) ([]byte, error) {
	const mebibyte = 1 * size.Mebibyte / size.Kibibyte

	return EncodedHash(password, h.params)
}

func (h *Hasher) Verify(password, encodedHash []byte) (bool, bool, error) {
	return Verify(password, encodedHash, &h.params)
}
