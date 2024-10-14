package system

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type UpdateEmailsGuard interface {
	CanUpdateEmails() bool
}

type UpdateEmailsInput struct {
	SystemEmail   Email
	SecurityEmail Email
}

func (s *Service) UpdateEmailsValidate(guard UpdateEmailsGuard, systemEmail, securityEmail string) (UpdateEmailsInput, error) {
	var input UpdateEmailsInput

	if !guard.CanUpdateEmails() {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	if input.SystemEmail, err = NewEmail(systemEmail); err != nil {
		errs.Set("system email", err)
	}
	if input.SecurityEmail, err = NewEmail(securityEmail); err != nil {
		errs.Set("security email", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) UpdateEmails(ctx context.Context, guard UpdateEmailsGuard, systemEmail, securityEmail string) (*Config, error) {
	input, err := s.UpdateEmailsValidate(guard, systemEmail, securityEmail)
	if err != nil {
		return nil, err
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	config.ChangeSystemEmail(input.SystemEmail)
	config.ChangeSecurityEmail(input.SecurityEmail)

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(&config.Events)

	return config, nil
}
