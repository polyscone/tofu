package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/repository"
)

var SuperRole *Role

type Reader interface {
	FindRoleByID(ctx context.Context, id int) (*Role, error)
	FindRoleByName(ctx context.Context, name string) (*Role, error)

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
}

type Service struct {
	broker event.Broker
	repo   ReadWriter
	hasher Hasher
}

func NewService(broker event.Broker, repo ReadWriter, hasher Hasher) (*Service, error) {
	var permissions []Permission
	for _, group := range guard.PermissionGroups {
		for _, p := range group.Permissions {
			permission, err := NewPermission(p.Name)
			if err != nil {
				return nil, fmt.Errorf("new permission: %w", err)
			}

			permissions = append(permissions, permission)
		}
	}

	SuperRole = NewRole("Super", "Has full access to the system; can't be edited or deleted.", permissions)

	ctx := context.Background()
	role, err := repo.FindRoleByName(ctx, SuperRole.Name)
	switch {
	case err == nil:
		SuperRole.ID = role.ID

		if err := repo.SaveRole(ctx, SuperRole); err != nil {
			return nil, fmt.Errorf("save super role: %w", err)
		}

	case errors.Is(err, repository.ErrNotFound):
		if err := repo.AddRole(ctx, SuperRole); err != nil {
			return nil, fmt.Errorf("add super role: %w", err)
		}

	default:
		return nil, fmt.Errorf("find role by name: %w", err)
	}

	svc := Service{
		broker: broker,
		repo:   repo,
		hasher: hasher,
	}

	return &svc, nil
}
