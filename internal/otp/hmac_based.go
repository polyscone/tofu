package otp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
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
		return otp, errors.New("invalid algorithm provided")
	}

	if min := 6; digits < min {
		return otp, fmt.Errorf("digits must be at least %v; got %v", min, digits)
	}

	otp.digits = digits

	return otp, nil
}

func (otp HMACBased) Generate(key []byte, count uint64) (string, error) {
	if n := len(key); n < otp.minKeyLen {
		return "", fmt.Errorf("key must be at least %v bytes; got %v", otp.minKeyLen, n)
	}

	mac := hmac.New(otp.newHash, key)
	if err := binary.Write(mac, binary.BigEndian, count); err != nil {
		return "", fmt.Errorf("write binary: %w", err)
	}

	sum := mac.Sum(nil)

	truncated := otp.truncate(sum)
	zeroPadded := fmt.Sprintf("%0*d", otp.digits, truncated)

	return zeroPadded, nil
}

func (otp HMACBased) truncate(sum []byte) uint {
	sbits := dt(sum)

	var snum uint
	snum |= uint(sbits[0]) << 24
	snum |= uint(sbits[1]) << 16
	snum |= uint(sbits[2]) << 8
	snum |= uint(sbits[3])

	return snum % uint(math.Pow(10, float64(otp.digits)))
}

func dt(sum []byte) []byte {
	// The offset is always the 4 least significant bits of the least significant
	// byte, assuming little endian
	offset := sum[len(sum)-1] & 0b0000_1111

	// p is made up of the 4 bytes starting at offset
	p := sum[offset : offset+4]

	// We need to return the least significant 31 bits of p, so we'll just mask
	// out the most significant bit
	p[0] &= 0b0111_1111

	return p
}
