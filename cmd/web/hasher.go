package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/polyscone/tofu/internal/pkg/password/argon2"
	"github.com/polyscone/tofu/internal/pkg/size"
	"golang.org/x/exp/slog"
)

var hasher *Hasher

type Hasher struct {
	params argon2.Params
}

func (h *Hasher) EncodedPasswordHash(password []byte) ([]byte, error) {
	return argon2.EncodedHash(nil, password, h.params)
}

func (h *Hasher) CheckPasswordHash(password, encodedHash []byte) (ok, rehash bool, _ error) {
	return argon2.Check(password, encodedHash, &h.params)
}

func initHasher() error {
	var params argon2.Params

	paramsCache := filepath.Join(opts.data, "argon2_params.json")
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

		params, _ = argon2.Calibrate(1*time.Second, argon2.ID, 64*size.Mebibyte, runtime.NumCPU()*2)
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

	hasher = &Hasher{params: params}

	return nil
}
