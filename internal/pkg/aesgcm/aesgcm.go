package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

// Encrypt will encrypt the given plaintext with the given key and return the
// final ciphertext or an error.
//
// The key can be of any length and will be turned into a SHA256 hash before
// encryption of the plaintext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(key); err != nil {
		return nil, errors.Tracef(err)
	}
	key = h.Sum(nil)

	return EncryptWithKey(key, plaintext)
}

// EncryptWithKey will encrypt the given plaintext with the given key and return
// the final ciphertext or an error.
//
// The key must be 32 bytes in length otherwise an error will be returned.
// To use a key of any length use the Encrypt function instead.
func EncryptWithKey(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.Tracef("want 256 bit key, got %d bit key", len(key)*8)
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Tracef(err)
	}

	// We set the nonce slice as both the dst and nonce args here so that
	// the encrypted plaintext will be appended to the nonce
	// This means that after this call nonce and ciphertext are actually the
	// same slice
	// We do this so that we can slice the nonce off the beginning of the
	// encrypted data for decryption
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt will decrypt the given ciphertext with the given key and return the
// original plaintext or an error.
//
// The key can be of any length and will be turned into a SHA256 hash before
// decryption of the ciphertext.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(key); err != nil {
		return nil, errors.Tracef(err)
	}
	key = h.Sum(nil)

	return DecryptWithKey(key, ciphertext)
}

// DecryptWithKey will decrypt the given ciphertext with the given key and return
// the original plaintext or an error.
//
// The key must be 32 bytes in length otherwise an error will be returned.
// To use a key of any length use the Decrypt function instead.
func DecryptWithKey(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.Tracef("want 256 bit key, got %d bit key", len(key)*8)
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	// We expect the nonce to be prepended on the given ciphertext, so we need
	// to slice it off and reassign the ciphertext slice to the correct position
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	return plaintext, nil
}
