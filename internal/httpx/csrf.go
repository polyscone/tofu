package httpx

import (
	"context"
	"encoding/base64"

	"github.com/polyscone/tofu/internal/csrf"
)

func MaskedCSRFToken(ctx context.Context) string {
	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
}
