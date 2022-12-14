package otp

import (
	"crypto/rand"
	"io"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

type Alg int

const (
	SHA1   Alg = 1
	SHA512 Alg = 2
)

// NewKey will use the given reader to generate some random bytes to be used
// as the key in a one time password.
//
// Because of this it's important that the reader be set to nil in a production
// environment so that internally the function can use the most secure option
// which will be the standard library's crypto/rand reader.
func NewKey(r io.Reader, alg Alg) ([]byte, error) {
	var n int
	switch alg {
	case SHA1:
		n = 20

	case SHA512:
		n = 64

	default:
		return nil, errors.Tracef("new key requires a valid algorithm")
	}

	if r == nil {
		r = rand.Reader
	}

	b := make([]byte, n)
	_, err := io.ReadFull(r, b)

	return b, errors.Tracef(err)
}

// NewRecoveryCode will use the given reader to generate some random bytes to
// be used as a recovery code.
//
// Because of this it's important that the reader be set to nil in a production
// environment so that internally the function can use the most secure option
// which will be the standard library's crypto/rand reader.
func NewRecoveryCode(r io.Reader) ([]byte, error) {
	if r == nil {
		r = rand.Reader
	}

	b := make([]byte, 8)
	_, err := io.ReadFull(r, b)

	return b, errors.Tracef(err)
}
