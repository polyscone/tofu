package token

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type Repo interface {
	AddActivationToken(ctx context.Context, email text.Email, ttl time.Duration) (string, error)
	Consume(ctx context.Context, token string) (text.Email, error)
}
