package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

type UserRepo interface {
	FindByID(ctx context.Context, id uuid.V4) (User, error)
	FindByEmail(ctx context.Context, email text.Email) (User, error)
	Add(ctx context.Context, u User) error
	Save(ctx context.Context, u User) error
}
