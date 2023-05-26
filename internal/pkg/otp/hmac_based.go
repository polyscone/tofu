package otp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"hash"
	"math"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

// HMACBased implements an "HMAC-based One Time Password" (HOTP) in accordance
// with RFC4226 and any errata.
type HMACBased struct {
	digits    int
	newHash   func() hash.Hash
	minKeyLen int
}

func NewHMACBased(digits int, alg Algorithm) (HMACBased, error) {
	var otp HMACBased

	switch alg {
	case SHA1:
		otp.newHash = sha1.New
		otp.minKeyLen = 20

	case SHA512:
		otp.newHash = sha512.New
		otp.minKeyLen = 64

	default:
		return otp, errors.Tracef("invalid algorithm provided")
	}

	if min := 6; digits < min {
		return otp, errors.Tracef("digits must be at least %v; got %v", min, digits)
	}

	otp.digits = digits

	return otp, nil
}

func (otp HMACBased) Generate(key []byte, count uint64) (string, error) {
	if len(key) < otp.minKeyLen {
		return "", errors.Tracef("key must be at least %d bytes; got %d", otp.minKeyLen, key)
	}

	h := hmac.New(otp.newHash, key)
	if err := binary.Write(h, binary.BigEndian, count); err != nil {
		return "", errors.Tracef(err)
	}

	hs := h.Sum(nil)

	truncated := otp.truncate(hs)
	zeroPadded := fmt.Sprintf("%0*d", otp.digits, truncated)

	return zeroPadded, nil
}

func (otp HMACBased) truncate(hs []byte) uint {
	sbits := dt(hs)

	var snum uint
	snum |= uint(sbits[0]) << 24
	snum |= uint(sbits[1]) << 16
	snum |= uint(sbits[2]) << 8
	snum |= uint(sbits[3])

	return snum % uint(math.Pow(10, float64(otp.digits)))
}

func dt(hs []byte) []byte {
	// The offset is always the 4 lest significant bits of the lest significant
	// byte, assuming little endian
	offset := hs[len(hs)-1] & 0b0000_1111

	// p is made up of the 4 bytes starting at offset
	p := hs[offset : offset+4]

	// We need to return the lest significant 31 bits of p, so we'll just mask
	// out the most significant bit
	p[0] &= 0b0111_1111

	return p
}
