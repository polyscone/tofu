package account

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/repo"
)

var SuperRole *Role

type Reader interface {
	FindRoleByID(ctx context.Context, id int) (*Role, error)
	FindRoleByName(ctx context.Context, name string) (*Role, error)

	CountUsersByRoleID(ctx context.Context, roleID int) (int, error)
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
	store  ReadWriter
	hasher password.Hasher
}

func NewService(broker event.Broker, store ReadWriter, hasher password.Hasher) (*Service, error) {
	var permissions []Permission
	for _, group := range guard.PermissionGroups {
		for _, p := range group.Permissions {
			permission, err := NewPermission(p.Name)
			if err != nil {
				return nil, errors.Tracef(err)
			}

			permissions = append(permissions, permission)
		}
	}

	SuperRole = NewRole("Super", "Has full access to the system; cannot be edited or deleted.", permissions)

	ctx := context.Background()
	role, err := store.FindRoleByName(ctx, SuperRole.Name)
	switch {
	case err == nil:
		SuperRole.ID = role.ID

		if err := store.SaveRole(ctx, SuperRole); err != nil {
			return nil, errors.Tracef(err)
		}

	case errors.Is(err, repo.ErrNotFound):
		if err := store.AddRole(ctx, SuperRole); err != nil {
			return nil, errors.Tracef(err)
		}

	default:
		return nil, errors.Tracef(err)
	}

	svc := Service{
		broker: broker,
		store:  store,
		hasher: hasher,
	}

	return &svc, nil
}
