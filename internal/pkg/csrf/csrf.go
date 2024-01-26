package csrf

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
)

type ctxKey byte

const tokenDataKey ctxKey = iota

const tokenLength = 32

var (
	ErrEmptyToken   = errors.New("empty CSRF token")
	ErrInvalidToken = errors.New("invalid CSRF token")
)

type csrf struct {
	token []byte
	isNew bool
}

// IsNew returns true if the CSRF data on the given context has either been
// newly created or renewed.
func IsNew(ctx context.Context) bool {
	data := getCSRF(ctx)

	return data.isNew
}

// MaskedToken returns the CSRF token on the given context but masks it using
// a one time pad every time it's called.
//
// This means that the token returned will look different every time the
// function is called, but will produce the same value when XOR'ed with the key,
// which is prepended to the data as the first half of the byte slice.
//
// This is purely to help mitigate against things like the BREACH attack and a new
// CSRF token should still be generated on events like auth changes.
func MaskedToken(ctx context.Context) []byte {
	data := getCSRF(ctx)
	masked, err := mask(data.token)
	if err != nil {
		panic(err)
	}

	return masked
}

// RenewToken generates a new CSRF token and replaces it on the given context.
func RenewToken(ctx context.Context) error {
	token, err := newToken()
	if err != nil {
		return fmt.Errorf("new token: %w", err)
	}

	data := getCSRF(ctx)

	copy(data.token, token)

	data.isNew = true

	return nil
}

// SetToken accepts a masked CSRF token to set on the given context.
// If no token is provided then a new one is automatically generated
// and used instead.
func SetToken(ctx context.Context, masked []byte) (context.Context, error) {
	var unmasked []byte
	if masked == nil {
		token, err := newToken()
		if err != nil {
			return ctx, fmt.Errorf("new token: %w", err)
		}

		unmasked = token
	} else {
		if want, got := tokenLength*2, len(masked); want != got {
			return ctx, fmt.Errorf("masked token must be %v bytes in length; got %v", want, got)
		}

		token, err := unmask(masked)
		if err != nil {
			return ctx, fmt.Errorf("unmask: %w", err)
		}

		unmasked = token
	}

	data := csrf{
		token: unmasked,
		isNew: masked == nil,
	}

	return context.WithValue(ctx, tokenDataKey, &data), nil
}

// Check accepts a masked token to compare with the one on the given context.
// If the tokens match then it returns nil.
func Check(ctx context.Context, maskedCmp []byte) error {
	if maskedCmp == nil {
		return ErrEmptyToken
	}

	unmasked, err := unmask(MaskedToken(ctx))
	if err != nil {
		return fmt.Errorf("%w: unmask: %w", ErrInvalidToken, err)
	}

	unmaskedCmp, err := unmask(maskedCmp)
	if err != nil {
		return fmt.Errorf("%w: unmask comparison: %w", ErrInvalidToken, err)
	}

	if subtle.ConstantTimeCompare(unmasked, unmaskedCmp) != 1 {
		return ErrInvalidToken
	}

	return nil
}

func getCSRF(ctx context.Context) *csrf {
	value := ctx.Value(tokenDataKey)
	if value == nil {
		ctx, err := SetToken(ctx, nil)
		if err != nil {
			panic(err)
		}

		return getCSRF(ctx)
	}

	data, ok := value.(*csrf)
	if !ok {
		panic(fmt.Sprintf("could not assert token data as %T", data))
	}

	return data
}

func mask(token []byte) ([]byte, error) {
	n := len(token)
	masked := make([]byte, n*2)
	key := masked[:n]
	data := masked[n:]

	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}

	for i, b := range token {
		data[i] = b ^ key[i]
	}

	return masked, nil
}

func unmask(masked []byte) ([]byte, error) {
	if len(masked)%2 != 0 {
		return nil, errors.New("masked token must be an even length")
	}

	n := len(masked) / 2
	unmasked := make([]byte, n)
	key := masked[:n]
	data := masked[n:]

	for i, b := range data {
		unmasked[i] = b ^ key[i]
	}

	return unmasked, nil
}

func newToken() ([]byte, error) {
	b := make([]byte, tokenLength)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}

	return b, nil
}
