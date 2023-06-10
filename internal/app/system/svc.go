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
	store  ReadWriter
}

func NewService(broker event.Broker, store ReadWriter) (*Service, error) {
	svc := Service{
		broker: broker,
		store:  store,
	}

	return &svc, nil
}
