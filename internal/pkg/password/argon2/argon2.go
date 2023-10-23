package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/size"
	"golang.org/x/crypto/argon2"
)

// Variant represents the variants of an Argon2 hash.
type Variant string

// These are the available Argon2 variants.
// If you have the choice then Argon2id is recommended.
const (
	I  Variant = "argon2i"
	ID Variant = "argon2id"
)

// Params holds the parameters that will be used in the Argon2 key
// derivation functions.
//
// A sensible starting point for Argon2id would be to set Time to
// 1, and Memory to 64 MiB (64 * 1024 KiB).
//
// For Argon2i a sensible starting point would be to set Time to
// 3, and Memory to 32 MiB (32 * 1024 KiB).
//
// Argon2 key derivation functions expect the memory parameter to be expressed
// in terms of kibibytes (KiB).
// That is, a memory value of 1024 is actually 1024 KiB, not 1024 bytes, as
// you might expect.
//
// For example, if the memory should be set to 64 MiB then the memory
// field should be set to 65536.
// This is because 65536 KiB is the same as 64 MiB (65536 / 1024 = 64).
//
// The parallelism parameter sets the number of threads that will be used to
// spread the work across.
// Changing this parameter will also change the final output
// of the encoded hash.
//
// So even if all other parameters remain the same, just spreading the work
// across multiple threads will result in completely different output.
//
// For more information see: https://golang.org/x/crypto/argon2
type Params struct {
	Variant     Variant
	Time        uint32
	Memory      uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// IsValid will check to see if the current parameters are valid for use or not.
// If they are not valid then an error will be returned.
//
// It's important to note that just because this function may return true, it
// does not mean that the given parameters are actually suitable for hashing
// an actual password.
//
// The limits tested in this method are deliberately low so as
// to provide maximum flexibility.
func (p *Params) IsValid() error {
	if p.Variant != I && p.Variant != ID {
		return fmt.Errorf("unknown variant %q", p.Variant)
	}
	if want := uint32(1); p.Time < want {
		return fmt.Errorf("time must be %d or above", want)
	}
	if want := uint32(size.Kibibyte); p.Memory < want {
		// Memory is a minimum of 1 MiB, which is 1024 KiB
		return fmt.Errorf("memory must be %d or above", want)
	}
	if want := uint8(1); p.Parallelism < want {
		return fmt.Errorf("parallelism must be %d or above", want)
	}
	if want := uint32(8); p.SaltLength < want {
		return fmt.Errorf("salt length must be %d or above", want)
	}
	if want := uint32(16); p.KeyLength < want {
		return fmt.Errorf("key length must be %d or above", want)
	}
	return nil
}

// Calibrate starts with a set of minimum hashing parameters and increases them until
// it hits the desired target duration.
//
// The amount of memory should be expressed as a number of kibibytes (KiB).
// For example, if the memory should be set to 64 MiB then the memory
// parameter should be set to 65536.
// This is because 65536 KiB is the same as 64 MiB (65536 / 1024 = 64).
func Calibrate(target time.Duration, variant Variant, memory, parallelism int) (Params, time.Duration) {
	if memory <= 0 {
		panic("memory must be set")
	}
	if parallelism <= 0 {
		panic("parallelism must be set")
	}

	var t uint32
	switch variant {
	case I:
		t = 3

	case ID:
		t = 1

	default:
		panic(fmt.Sprintf("unknown variant %q", variant))
	}

	params := Params{
		Variant:     variant,
		Time:        t,
		Memory:      uint32(memory),
		Parallelism: uint8(parallelism),
		SaltLength:  16,
		KeyLength:   32,
	}

	password := make([]byte, int(params.KeyLength))
	if _, err := io.ReadFull(rand.Reader, password); err != nil {
		panic(err)
	}

	salt := make([]byte, int(params.SaltLength))
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic(err)
	}

CalibrateLoop:
	for {
		t := time.Now()
		key(password, salt, params)
		took := time.Since(t)

		if took >= target {
			// Double check the time taken just in case we need to
			// increase parameter values again
			t := time.Now()
			key(password, salt, params)
			if took := time.Since(t); took < target {
				continue CalibrateLoop
			}

			return params, took
		}

		params.Time++
	}
}

func key(password, salt []byte, p Params) {
	switch p.Variant {
	case I:
		argon2.Key(password, salt, p.Time, p.Memory, p.Parallelism, p.KeyLength)

	case ID:
		argon2.IDKey(password, salt, p.Time, p.Memory, p.Parallelism, p.KeyLength)

	default:
		panic(fmt.Sprintf("unknown variant %q", p.Variant))
	}
}

// encodedHashWithSalt will generate and return an encoded variant of an Argon2
// hash based on the given parameters.
//
// The encoded has returned will follow the format:
// $argon2x$v=19$m=65536,t=1,p=1$salt$key.
//
// The salt and key will be base64 encoded.
func encodedHashWithSalt(password, salt []byte, p Params) ([]byte, error) {
	if err := p.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	var key []byte
	switch p.Variant {
	case I:
		key = argon2.Key(password, salt, p.Time, p.Memory, p.Parallelism, p.KeyLength)

	case ID:
		key = argon2.IDKey(password, salt, p.Time, p.Memory, p.Parallelism, p.KeyLength)

	default:
		return nil, fmt.Errorf("unknown variant %q", p.Variant)
	}

	// Right now the salt and key values are just a slice of bytes, so we need
	// to encode them in a way that will be easy for a user of the function to
	// store somewhere for later
	//
	// The standard way of doing this is to use a base64 encoding without padding
	// In the Go standard library the StdEncoding includes padding, so
	// we use RawStdEncoding instead
	base64Salt := base64.RawStdEncoding.EncodeToString(salt)
	base64Key := base64.RawStdEncoding.EncodeToString(key)

	// The final encoded hash will be built up in the standard format:
	// $argon2x$v=19$m=65536,t=1,p=1$salt$key
	encodedHash := fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		p.Variant, argon2.Version, p.Memory, p.Time, p.Parallelism, base64Salt, base64Key,
	)

	return []byte(encodedHash), nil
}

// EncodedHash will generate and return an encoded variant of an Argon2 hash
// based on the given parameters.
//
// The encoded hash returned will follow the format:
// $argon2x$v=19$m=65536,t=1,p=1$salt$key.
//
// The salt and key will be base64 encoded and the salt will be
// generated using a CSrand.
func EncodedHash(r io.Reader, password []byte, p Params) ([]byte, error) {
	if r == nil {
		r = rand.Reader
	}

	salt := make([]byte, int(p.SaltLength))
	if _, err := io.ReadFull(r, salt); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}

	return encodedHashWithSalt(password, salt, p)
}

// Check will check to see whether the given password matches the given encoded
// hash or not.
//
// If a preferred argument is provided then the rehash return value will be
// set based on whether any of those parameters are different from the encoded
// hash's because preferred is treated as the "preferred" parameters.
//
// The rehash return value will only be set to anything other than false
// on a successful check.
func Check(password, encodedHash []byte, preferred *Params) (bool, bool, error) {
	var isValid bool
	var rehash bool

	parts := strings.Split(string(encodedHash), "$")
	if want := 6; len(parts) != want {
		return isValid, rehash, fmt.Errorf("invalid encoded hash, want %d parts; got %d", want, len(parts))
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return isValid, rehash, fmt.Errorf("scan version: %w", err)
	}
	if version != argon2.Version {
		// If the version of Argon2 in the package we're using is different from
		// that of the encoded hash we need to compare with then we should error
		// out, because we can't compare correctly in this case
		return isValid, rehash, fmt.Errorf("want version %d; got %d", argon2.Version, version)
	}

	// The salt in the encoded hash is base64 encoded, so we need to decode it
	// in order to get the correct salt length
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return isValid, rehash, fmt.Errorf("base64 decode salt: %w", err)
	}

	// The key in the encoded hash is base64 encoded, so we need to decode it
	// in order to get the correct key length
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return isValid, rehash, fmt.Errorf("base64 decode key: %w", err)
	}
	p := Params{
		Variant:    Variant(parts[1]),
		SaltLength: uint32(len(salt)),
		KeyLength:  uint32(len(key)),
	}

	// Extract the memory, time (time), and parallelism parameters from
	// the encoded hash we need to compare with
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Parallelism); err != nil {
		return isValid, rehash, fmt.Errorf("scan memory, time, and parallelism: %w", err)
	}

	encodedPassword, err := encodedHashWithSalt(password, salt, p)
	if err != nil {
		return isValid, rehash, fmt.Errorf("encode hash with salt: %w", err)
	}

	isValid = subtle.ConstantTimeCompare(encodedPassword, encodedHash) == 1

	if isValid && preferred != nil {
		// If the preferred parameters that were passed in aren't valid then
		// we'll return the isValid value as it was evaluated from the
		// comparison of the two hashes, but we'll also return the error at the
		// same time
		// The user of the function shouldn't really be ignoring the error, but
		// in the case that they do rehash will be false at this point anyway,
		// and the isValid value will at least correctly signal whether the
		// password and encoded hash to compare with were a match
		if err := preferred.IsValid(); err != nil {
			return isValid, rehash, fmt.Errorf("invalid params: %w", err)
		}

		// If any of the parameters passing into the function are different from
		// the parameters we extracted from the encoded hash we compared against
		// then it means we should signal that a rehash is needed
		// It's then up to the user of the function to decide whether to rehash
		// the password using their preferred parameters or not
		rehash = preferred.Variant != p.Variant ||
			preferred.Time != p.Time ||
			preferred.Memory != p.Memory ||
			preferred.Parallelism != p.Parallelism ||
			preferred.SaltLength != p.SaltLength ||
			preferred.KeyLength != p.KeyLength
	}

	return isValid, rehash, nil
}
