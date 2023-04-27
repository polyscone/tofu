package token

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type Repo interface {
	AddActivationToken(ctx context.Context, email text.Email, ttl time.Duration) (string, error)
	FindActivationTokenEmail(ctx context.Context, token string) (text.Email, error)
	ConsumeActivationToken(ctx context.Context, token string) error

	AddResetPasswordToken(ctx context.Context, email text.Email, ttl time.Duration) (string, error)
	FindResetPasswordTokenEmail(ctx context.Context, token string) (text.Email, error)
	ConsumeResetPasswordToken(ctx context.Context, token string) error
}
