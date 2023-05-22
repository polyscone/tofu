package query

import (
	"context"
	"time"
)

type TokenRepo interface {
	AddActivationToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindActivationTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeActivationToken(ctx context.Context, token string) error

	AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error)
	FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error)
	ConsumeResetPasswordToken(ctx context.Context, token string) error
}
