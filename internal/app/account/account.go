package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/app/account/internal/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

var NewSQLiteUserRepo = sqlite.NewUserRepo

type (
	Registered                = domain.Registered
	Verified                  = domain.Activated
	AuthenticatedWithPassword = domain.AuthenticatedWithPassword
	AuthenticatedWithTOTP     = domain.AuthenticatedWithTOTP
	ChangedPassword           = domain.ChangedPassword
)

type UserRepo interface {
	FindByID(ctx context.Context, id uuid.V4) (domain.User, error)
	FindByEmail(ctx context.Context, email text.Email) (domain.User, error)
	Add(ctx context.Context, u domain.User) error
	Save(ctx context.Context, u domain.User) error
}