package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/polyscone/tofu/internal/password/argon2"
)

var hasher *Hasher

type Hasher struct {
	params argon2.Params
	dummy  []byte
}

func (h *Hasher) EncodedPasswordHash(password []byte) ([]byte, error) {
	return argon2.EncodedHash(nil, password, h.params)
}

func (h *Hasher) CheckPasswordHash(password, encodedHash []byte) (ok, rehash bool, _ error) {
	return argon2.Check(password, encodedHash, &h.params)
}

func (h *Hasher) CheckDummyPasswordHash() error {
	_, _, err := argon2.Check([]byte("password"), h.dummy, &h.params)

	return err
}

func initHasher() error {
	var params argon2.Params

	paramsCache := filepath.Join(opts.dataDir, "argon2_params.json")
	cachedParams, err := os.ReadFile(paramsCache)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read argon2 params: %w", err)
	} else if err == nil {
		if err := json.Unmarshal(cachedParams, &params); err != nil {
			return fmt.Errorf("unmarshal argon2 params: %w", err)
		}
	}

	if params.IsValid() != nil {
		slog.Info("detecting new argon2 password hashing parameters, please wait...")

		params, _ = argon2.Calibrate(opts.password.duration, argon2.ID, opts.password.memory, opts.password.parallelism)
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal argon2 params: %w", err)
		}

		if err := os.WriteFile(paramsCache, paramsJSON, 0755); err != nil {
			return fmt.Errorf("write argon2 params: %w", err)
		}

		slog.Info("new argon2 password hashing parameters detected and cached", "location", paramsCache)
	}

	if err := params.IsValid(); err != nil {
		return fmt.Errorf("invalid argon2 params: %w", err)
	}

	dummy, err := argon2.EncodedHash(nil, []byte("correct horse battery staple"), params)
	if err != nil {
		return fmt.Errorf("new dummy hash: %w", err)
	}

	hasher = &Hasher{
		params: params,
		dummy:  dummy,
	}

	return nil
}
