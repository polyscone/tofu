package query

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/pkg/repo"
)

type AccountRole struct {
	ID          string
	Name        string
	Permissions []string
}

type AccountUser struct {
	ID            string
	Email         string
	TOTPUseSMS    bool
	TOTPTelephone string
	Claims        []string
	Roles         []AccountRole
	ActivatedAt   time.Time
}

type TOTPParams struct {
	Key       []byte
	Algorithm string
	Digits    int
	Period    time.Duration
}

type AccountUserRepo interface {
	FindByID(ctx context.Context, userID string) (AccountUser, error)
	FindByEmail(ctx context.Context, email string) (AccountUser, error)
	FindByPageFilter(ctx context.Context, page, size int, filter string) (*repo.Book[AccountUser], error)
	FindTOTPParamsByID(ctx context.Context, userID string) (TOTPParams, error)
	FindRecoveryCodesByID(ctx context.Context, userID string) ([]string, error)
}
