package argon2

import (
	"crypto/rand"
	"io"

	"github.com/polyscone/tofu/internal/pkg/size"
)

type Hasher struct {
	reader io.Reader
	params Params
}

func NewHasher(r io.Reader, params Params) *Hasher {
	if r == nil {
		r = rand.Reader
	}

	return &Hasher{
		reader: r,
		params: params,
	}
}

func (h *Hasher) EncodedHash(password []byte) ([]byte, error) {
	const mebibyte = 1 * size.Mebibyte / size.Kibibyte

	return EncodedHash(h.reader, password, h.params)
}

func (h *Hasher) Verify(password, encodedHash []byte) (bool, bool, error) {
	return Verify(password, encodedHash, &h.params)
}
