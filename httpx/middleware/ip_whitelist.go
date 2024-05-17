package middleware

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/realip"
)

type IPWhitelistConfig struct {
	IPs            []string
	TrustedProxies []string
	ErrorHandler   ErrorHandler
}

func IPWhitelist(config *IPWhitelistConfig) Middleware {
	if config == nil {
		config = &IPWhitelistConfig{}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip, err := realip.FromRequest(r, config.TrustedProxies)
			if err != nil {
				err = fmt.Errorf("realip from request: %w", err)

				handleError(w, r, err, config.ErrorHandler, http.StatusInternalServerError)

				return
			}

			if !slices.Contains(config.IPs, ip) {
				err := fmt.Errorf("%w: ip %v is not in the whitelist", httpx.ErrForbidden, ip)

				handleError(w, r, err, config.ErrorHandler, http.StatusForbidden)

				return
			}

			next(w, r)
		}
	}
}
