package system

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type UpdateConfigGuard interface {
	CanUpdateConfig() bool
}

func (s *Service) UpdateConfig(ctx context.Context, guard UpdateConfigGuard,
	systemEmail, securityEmail string,
	signUpEnabled bool,
	totpRequired bool,
	googleSignInEnabled bool, googleSignInClientID string,
	twilioSID, twilioToken, twilioFromTel string,
) (*Config, error) {
	var input struct {
		systemEmail          Email
		securityEmail        Email
		signUpEnabled        bool
		totpRequired         bool
		googleSignInEnabled  bool
		googleSignInClientID GoogleClientID
		twilioSID            TwilioSID
		twilioToken          TwilioToken
		twilioFromTel        TwilioTel
	}
	{
		if !guard.CanUpdateConfig() {
			return nil, app.ErrUnauthorised
		}

		var err error
		var errs errsx.Map

		if input.systemEmail, err = NewEmail(systemEmail); err != nil {
			errs.Set("system email", err)
		}
		if input.securityEmail, err = NewEmail(securityEmail); err != nil {
			errs.Set("security email", err)
		}

		input.signUpEnabled = signUpEnabled
		input.totpRequired = totpRequired
		input.googleSignInEnabled = googleSignInEnabled

		if input.googleSignInClientID, err = NewGoogleClientID(googleSignInClientID); err != nil {
			errs.Set("google sign in client id", err)
		}
		if input.twilioSID, err = NewTwilioSID(twilioSID); err != nil {
			errs.Set("twilio sid", err)
		}
		if input.twilioToken, err = NewTwilioToken(twilioToken); err != nil {
			errs.Set("twilio token", err)
		}
		if input.twilioFromTel, err = NewTwilioTel(twilioFromTel); err != nil {
			errs.Set("twilio from tel", err)
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

	if input.signUpEnabled {
		config.EnableSignUp()
	} else {
		config.DisableSignUp()
	}

	if input.totpRequired {
		config.EnableRequireTOTP()
	} else {
		config.DisableRequireTOTP()
	}

	config.ChangeGoogleSignInClientID(input.googleSignInClientID)
	if input.googleSignInEnabled {
		if err := config.EnableGoogleSignIn(); err != nil {
			return nil, fmt.Errorf("enable Google sign in: %w", err)
		}
	} else {
		config.DisableGoogleSignIn()
	}

	config.ChangeTwilioAPI(input.twilioSID, input.twilioToken, input.twilioFromTel)

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(&config.Events)

	return config, nil
}
