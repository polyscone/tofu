package system

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
)

type UpdateEmailsGuard interface {
	CanUpdateEmails() bool
}

func (s *Service) UpdateEmails(ctx context.Context, guard UpdateEmailsGuard, systemEmail, securityEmail string) error {
	var input struct {
		systemEmail   Email
		securityEmail Email
	}
	{
		if !guard.CanUpdateEmails() {
			return app.ErrForbidden
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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return fmt.Errorf("find config: %w", err)
	}

	config.ChangeSystemEmail(input.systemEmail)
	config.ChangeSecurityEmail(input.securityEmail)

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(&config.Events)

	return nil
}
