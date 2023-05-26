package otp

import (
	"crypto/rand"
	"io"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

const (
	Invalid Algorithm = iota
	SHA1
	SHA512
)

type Algorithm int

func NewAlgorithm(algorithm string) (Algorithm, error) {
	switch algorithm {
	case "SHA1":
		return SHA1, nil

	case "SHA512":
		return SHA512, nil

	default:
		return Invalid, errors.Tracef("invalid algorithm %q", algorithm)
	}
}

func (a Algorithm) String() string {
	switch a {
	case SHA1:
		return "SHA1"

	case SHA512:
		return "SHA512"

	default:
		return "invalid"
	}
}

// NewKey will use the given reader to generate some random bytes to be used
// as the key in a one time password.
//
// Because of this it's important that the reader be set to nil in a production
// environment so that internally the function can use the most secure option
// which will be the standard library's crypto/rand reader.
func NewKey(r io.Reader, alg Algorithm) ([]byte, error) {
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
