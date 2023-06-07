package handler

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/sms"
	"github.com/polyscone/tofu/internal/pkg/smtp"
)

type AccountReader interface {
	account.Reader

	FindRoles(ctx context.Context, sortTopID int) ([]*account.Role, int, error)
	FindRolesPageBySearch(ctx context.Context, sortTopID int, search string, page, size int) ([]*account.Role, int, error)
	FindUsersPageBySearch(ctx context.Context, sortTopID int, search string, page, size int) ([]*account.User, int, error)
}

type WebReadWriter interface {
	session.ReadWriter

	AddActivationToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindActivationTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeActivationToken(ctx context.Context, token string) error

	AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeResetPasswordToken(ctx context.Context, token string) error
}

type Repo struct {
	Account AccountReader
	Web     WebReadWriter
}

type Email struct {
	From   string
	Mailer smtp.Mailer
}

type SMS struct {
	IsConfigured bool
	From         string
	Messager     sms.Messager
}

type Tenant struct {
	Scheme   string
	Host     string
	Hostname string
	Port     string
	Dev      bool
	Insecure bool
	Proxies  []string
	Broker   event.Broker
	Email    Email
	SMS      SMS

	Account *account.Service

	Repo Repo
}
