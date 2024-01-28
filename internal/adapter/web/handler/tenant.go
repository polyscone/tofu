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

	FindRoles(ctx context.Context, sortTopID int) ([]*account.Role, int, error)
	FindRolesPageBySearch(ctx context.Context, sortTopID int, sorts []string, search string, page, size int) ([]*account.Role, int, error)

	FindUsersPageBySearch(ctx context.Context, sortTopID int, sorts []string, search string, page, size int) ([]*account.User, int, error)
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
	Dev      bool
	Insecure bool
	Proxies  []string
	Broker   event.Broker
	Email    smtp.Mailer
	Logger   *slog.Logger
	Metrics  *expvar.Map

	Svc  Svc
	Repo Repo
}
