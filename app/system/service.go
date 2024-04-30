package system

import "github.com/polyscone/tofu/event"

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
