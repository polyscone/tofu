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
	SystemEmail   string
	SecurityEmail string
}

type UpdateEmailsData struct {
	SystemEmail   Email
	SecurityEmail Email
}

func (s *Service) UpdateEmailsValidate(guard UpdateEmailsGuard, input UpdateEmailsInput) (UpdateEmailsData, error) {
	var data UpdateEmailsData

	if !guard.CanUpdateEmails() {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	if data.SystemEmail, err = NewEmail(input.SystemEmail); err != nil {
		errs.Set("system email", err)
	}
	if data.SecurityEmail, err = NewEmail(input.SecurityEmail); err != nil {
		errs.Set("security email", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) UpdateEmails(ctx context.Context, guard UpdateEmailsGuard, input UpdateEmailsInput) (*Config, error) {
	data, err := s.UpdateEmailsValidate(guard, input)
	if err != nil {
		return nil, err
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	config.ChangeSystemEmail(data.SystemEmail)
	config.ChangeSecurityEmail(data.SecurityEmail)

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(ctx, &config.Events)

	return config, nil
}
