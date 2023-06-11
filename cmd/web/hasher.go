package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/pkg/password/argon2"
)

var hasher password.Hasher

func initPasswordHasher() error {
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
		logger.Info.Println("detecting new argon2 password hashing parameters...")

		params, _ = argon2.DetectParams(1*time.Second, argon2.ID, 0, 0)
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal argon2 params: %w", err)
		}

		if err := os.WriteFile(paramsCache, paramsJSON, 0666); err != nil {
			return fmt.Errorf("write argon2 params: %w", err)
		}

		logger.Info.Printf("new argon2 password hashing parameters detected and cached in %v\n", paramsCache)
	}

	if err := params.IsValid(); err != nil {
		return fmt.Errorf("invalid argon2 params: %w", err)
	}

	hasher = argon2.NewHasher(nil, params)

	return nil
}
