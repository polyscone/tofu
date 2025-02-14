package main

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/smtp"
	"github.com/polyscone/tofu/repo"
)

type smtpConfig struct {
	system *repo.System
}

func (c *smtpConfig) Read(ctx context.Context) (*smtp.ClientConfig, error) {
	config, err := c.system.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	smtpClientConfig := smtp.ClientConfig{
		EnvelopeFrom: opts.smtp.envelopeFrom,
		ResendAPIKey: config.ResendAPIKey,
	}

	return &smtpClientConfig, nil
}
