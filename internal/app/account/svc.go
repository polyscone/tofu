package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/password"
)

type Reader interface {
	FindRoleByID(ctx context.Context, id int) (*Role, error)
	FindUserByID(ctx context.Context, id int) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
}

type Writer interface {
	AddUser(ctx context.Context, u *User) error
	SaveUser(ctx context.Context, u *User) error
}

type ReadWriter interface {
	Reader
	Writer
}

type Service struct {
	broker event.Broker
	repo   ReadWriter
	hasher password.Hasher
}

func NewService(broker event.Broker, repo ReadWriter, hasher password.Hasher) *Service {
	return &Service{
		broker: broker,
		repo:   repo,
		hasher: hasher,
	}
}
