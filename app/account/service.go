package account

import (
	"errors"

	"github.com/polyscone/tofu/internal/event"
)

var ErrAuth = errors.New("auth")

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
