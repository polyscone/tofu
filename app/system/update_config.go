package system

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type UpdateConfigGuard interface {
	CanUpdateConfig() bool
}

type UpdateConfigInput struct {
	SystemEmail               string
	SecurityEmail             string
	SignUpEnabled             bool
	SignUpAutoActivateEnabled bool
	TOTPRequired              bool
	TOTPSMSEnabled            bool
	MagicLinkSignInEnabled    bool
	GoogleSignInEnabled       bool
	GoogleSignInClientID      string
	FacebookSignInEnabled     bool
	FacebookSignInAppID       string
	FacebookSignInAppSecret   string
	ResendAPIKey              string
	TwilioSID                 string
	TwilioToken               string
	TwilioFromTel             string
}

type UpdateConfigData struct {
	SystemEmail               Email
	SecurityEmail             Email
	SignUpEnabled             bool
	SignUpAutoActivateEnabled bool
	TOTPRequired              bool
	TOTPSMSEnabled            bool
	MagicLinkSignInEnabled    bool
	GoogleSignInEnabled       bool
	GoogleSignInClientID      GoogleClientID
	FacebookSignInEnabled     bool
	FacebookSignInAppID       FacebookAppID
	FacebookSignInAppSecret   FacebookAppSecret
	ResendAPIKey              ResendAPIKey
	TwilioSID                 TwilioSID
	TwilioToken               TwilioToken
	TwilioFromTel             TwilioTel
}

func (s *Service) UpdateConfigValidate(guard UpdateConfigGuard, input UpdateConfigInput) (UpdateConfigData, error) {
	var data UpdateConfigData

	if !guard.CanUpdateConfig() {
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

	data.SignUpEnabled = input.SignUpEnabled
	data.SignUpAutoActivateEnabled = input.SignUpAutoActivateEnabled
	data.TOTPRequired = input.TOTPRequired
	data.TOTPSMSEnabled = input.TOTPSMSEnabled
	data.MagicLinkSignInEnabled = input.MagicLinkSignInEnabled

	data.GoogleSignInEnabled = input.GoogleSignInEnabled

	if data.GoogleSignInClientID, err = NewGoogleClientID(input.GoogleSignInClientID); err != nil {
		errs.Set("google sign in client id", err)
	}

	data.FacebookSignInEnabled = input.FacebookSignInEnabled

	if data.FacebookSignInAppID, err = NewFacebookAppID(input.FacebookSignInAppID); err != nil {
		errs.Set("facebook sign in app id", err)
	}
	if data.FacebookSignInAppSecret, err = NewFacebookAppSecret(input.FacebookSignInAppSecret); err != nil {
		errs.Set("facebook sign in app secret", err)
	}

	if data.ResendAPIKey, err = NewResendAPIKey(input.ResendAPIKey); err != nil {
		errs.Set("resend API key", err)
	}
	if data.TwilioSID, err = NewTwilioSID(input.TwilioSID); err != nil {
		errs.Set("twilio sid", err)
	}
	if data.TwilioToken, err = NewTwilioToken(input.TwilioToken); err != nil {
		errs.Set("twilio token", err)
	}
	if data.TwilioFromTel, err = NewTwilioTel(input.TwilioFromTel); err != nil {
		errs.Set("twilio from tel", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) UpdateConfig(ctx context.Context, guard UpdateConfigGuard, input UpdateConfigInput) (*Config, error) {
	data, err := s.UpdateConfigValidate(guard, input)
	if err != nil {
		return nil, err
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	var errs errsx.Slice

	config.ChangeSystemEmail(data.SystemEmail)
	config.ChangeSecurityEmail(data.SecurityEmail)

	if data.SignUpEnabled {
		config.EnableSignUp()
	} else {
		config.DisableSignUp()
	}

	if data.SignUpAutoActivateEnabled {
		config.EnableSignUpAutoActivate()
	} else {
		config.DisableSignUpAutoActivate()
	}

	if data.TOTPRequired {
		config.EnableTOTPRequired()
	} else {
		config.DisableTOTPRequired()
	}

	if data.MagicLinkSignInEnabled {
		config.EnableMagicLinkSignIn()
	} else {
		config.DisableMagicLinkSignIn()
	}

	config.ChangeGoogleSignInClientID(data.GoogleSignInClientID)

	if data.GoogleSignInEnabled {
		errs.Append(config.EnableGoogleSignIn())
	} else {
		config.DisableGoogleSignIn()
	}

	config.ChangeFacebookSignInAppID(data.FacebookSignInAppID)
	config.ChangeFacebookSignInAppSecret(data.FacebookSignInAppSecret)

	if data.FacebookSignInEnabled {
		errs.Append(config.EnableFacebookSignIn())
	} else {
		config.DisableFacebookSignIn()
	}

	config.ChangeResendAPI(data.ResendAPIKey)
	config.ChangeTwilioAPI(data.TwilioSID, data.TwilioToken, data.TwilioFromTel)

	if data.TOTPSMSEnabled {
		errs.Append(config.EnableTOTPSMS())
	} else {
		config.DisableTOTPSMS()
	}

	if errs != nil {
		return nil, errs
	}

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(ctx, &config.Events)

	return config, nil
}
