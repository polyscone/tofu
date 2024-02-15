package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/event"
)

type Reader interface {
	FindRoleByID(ctx context.Context, id int) (*Role, error)
	FindRoleByName(ctx context.Context, name string) (*Role, error)

	CountUsers(ctx context.Context) (int, error)
	CountUsersByRoleID(ctx context.Context, roleID int) (int, error)
	FindUserByID(ctx context.Context, id int) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)

	FindSignInAttemptLogByEmail(ctx context.Context, email string) (*SignInAttemptLog, error)
}

type Writer interface {
	AddRole(ctx context.Context, role *Role) error
	SaveRole(ctx context.Context, role *Role) error
	RemoveRole(ctx context.Context, roleID int) error

	AddUser(ctx context.Context, user *User) error
	SaveUser(ctx context.Context, user *User) error

	SaveSignInAttemptLog(ctx context.Context, log *SignInAttemptLog) error
}

type ReadWriter interface {
	Reader
	Writer
}

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
