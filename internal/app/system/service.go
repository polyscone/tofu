package system

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/event"
)

type Reader interface {
	FindConfig(ctx context.Context) (*Config, error)
}

type Writer interface {
	SaveConfig(ctx context.Context, config *Config) error
}

type ReadWriter interface {
	Reader
	Writer
}

type Service struct {
	broker event.Broker
	repo   ReadWriter
}

func NewService(broker event.Broker, repo ReadWriter) (*Service, error) {
	svc := Service{
		broker: broker,
		repo:   repo,
	}

	return &svc, nil
}
