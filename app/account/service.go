package account

import "github.com/polyscone/tofu/event"

type Hasher interface {
	EncodedPasswordHash(password []byte) ([]byte, error)
	CheckPasswordHash(password, encodedHash []byte) (ok, rehash bool, err error)
	CheckDummyPasswordHash() error
}

type Service struct {
	broker event.Broker
	repo   ReadWriter
	hasher Hasher
	system string
}

func NewService(broker event.Broker, repo ReadWriter, hasher Hasher, system string) (*Service, error) {
	svc := Service{
		broker: broker,
		repo:   repo,
		hasher: hasher,
		system: system,
	}

	return &svc, nil
}
