package handler

import (
	"context"
	"expvar"
	"log/slog"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/smtp"
)

type AccountReader interface {
	account.Reader

	FindRoleByName(ctx context.Context, name string) (*account.Role, error)
	FindRoles(ctx context.Context, sortTopID string) ([]*account.Role, int, error)
	FindRolesPageBySearch(ctx context.Context, sortTopID string, sorts []string, search string, page, size int) ([]*account.Role, int, error)

	CountUsersByRoleID(ctx context.Context, roleID string) (int, error)
	FindUsersPageBySearch(ctx context.Context, sortTopID string, sorts []string, search string, page, size int) ([]*account.User, int, error)
}

type SystemReader system.Reader

type WebReadWriter interface {
	session.ReadWriter

	AddEmailVerificationToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindEmailVerificationTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeEmailVerificationToken(ctx context.Context, token string) error

	AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeResetPasswordToken(ctx context.Context, token string) error

	AddSignInMagicLinkToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindSignInMagicLinkTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeSignInMagicLinkToken(ctx context.Context, token string) error

	AddTOTPResetVerifyToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindTOTPResetVerifyTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeTOTPResetVerifyToken(ctx context.Context, token string) error

	AddResetTOTPToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindResetTOTPTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeResetTOTPToken(ctx context.Context, token string) error
}

type Svc struct {
	Account *account.Service
	System  *system.Service
}

type Repo struct {
	Account AccountReader
	System  SystemReader
	Web     WebReadWriter
}

type Tenant struct {
	Key      string
	Kind     string
	Scheme   string
	Host     string
	Hosts    map[string]string
	Data     string
	Dev      bool
	Insecure bool
	Proxies  []string
	Broker   event.Broker
	Email    smtp.Mailer
	Logger   *slog.Logger
	Metrics  *expvar.Map

	Svc       Svc
	Repo      Repo
	SuperRole *account.Role
}
