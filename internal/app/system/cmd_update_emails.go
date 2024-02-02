package system

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type UpdateEmailsGuard interface {
	CanUpdateEmails() bool
}

func (s *Service) UpdateEmails(ctx context.Context, guard UpdateEmailsGuard, systemEmail, securityEmail string) (*Config, error) {
	var input struct {
		systemEmail   Email
		securityEmail Email
	}
	{
		if !guard.CanUpdateEmails() {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.systemEmail, err = NewEmail(systemEmail); err != nil {
			errs.Set("system email", err)
		}
		if input.securityEmail, err = NewEmail(securityEmail); err != nil {
			errs.Set("security email", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	config.ChangeSystemEmail(input.systemEmail)
	config.ChangeSecurityEmail(input.securityEmail)

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(&config.Events)

	return config, nil
}
