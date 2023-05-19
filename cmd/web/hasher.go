package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
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
		return errors.Tracef(err)
	} else if err == nil {
		if err := json.Unmarshal(cachedParams, &params); err != nil {
			return errors.Tracef(err)
		}
	}

	if params.IsValid() != nil {
		params, _ = argon2.DetectParams(1*time.Second, argon2.ID, 0, 0)
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := os.WriteFile(paramsCache, paramsJSON, os.ModePerm); err != nil {
			return errors.Tracef(err)
		}

		logger.Info.Printf("new argon2 password hashing parameters detected and cached in %v\n", paramsCache)
	}

	if err := params.IsValid(); err != nil {
		return errors.Tracef(err)
	}

	hasher = argon2.NewHasher(params)

	return nil
}
