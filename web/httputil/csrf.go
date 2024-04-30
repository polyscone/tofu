package httputil

import (
	"context"
	"encoding/base64"

	"github.com/polyscone/tofu/csrf"
)

func MaskedCSRFToken(ctx context.Context) string {
	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
}
