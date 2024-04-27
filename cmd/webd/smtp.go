package main

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/pkg/smtp"
	"github.com/polyscone/tofu/sqlite"
)

type smtpConfig struct {
	system *sqlite.SystemRepo
}

func (c *smtpConfig) Read(ctx context.Context) (*smtp.ClientConfig, error) {
	config, err := c.system.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	return &smtp.ClientConfig{ResendAPIKey: config.ResendAPIKey}, nil
}
