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
	AddRole(ctx context.Context, role *Role) error
	SaveRole(ctx context.Context, role *Role) error
	RemoveRole(ctx context.Context, roleID int) error

	AddUser(ctx context.Context, user *User) error
	SaveUser(ctx context.Context, user *User) error
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
