package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account/domain"
	"github.com/polyscone/tofu/internal/port/account/repo/sqlite"
)

var NewSQLiteUserRepo = sqlite.NewUserRepo

type (
	Registered                    = domain.Registered
	Activated                     = domain.Activated
	AuthenticatedWithPassword     = domain.AuthenticatedWithPassword
	AuthenticatedWithTOTP         = domain.AuthenticatedWithTOTP
	AuthenticatedWithRecoveryCode = domain.AuthenticatedWithRecoveryCode
	DisabledTOTP                  = domain.DisabledTOTP
	RecoveryCodesRegenerated      = domain.RecoveryCodesRegenerated
	PasswordChanged               = domain.PasswordChanged
	PasswordReset                 = domain.PasswordReset
)

type UserRepo interface {
	FindByID(ctx context.Context, id uuid.V4) (domain.User, error)
	FindByEmail(ctx context.Context, email text.Email) (domain.User, error)
	Add(ctx context.Context, u domain.User) error
	Save(ctx context.Context, u domain.User) error
}
