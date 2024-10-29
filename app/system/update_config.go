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

func (s *Service) UpdateConfigValidate(
	guard UpdateConfigGuard,
	systemEmail, securityEmail string,
	signUpEnabled, signUpAutoActivateEnabled bool,
	totpRequired, totpSMSEnabled bool,
	magicLinkSignInEnabled bool,
	googleSignInEnabled bool, googleSignInClientID string,
	facebookSignInEnabled bool, facebookSignInAppID, facebookSignInAppSecret string,
	resendAPIKey string,
	twilioSID, twilioToken, twilioFromTel string,
) (UpdateConfigInput, error) {
	var input UpdateConfigInput

	if !guard.CanUpdateConfig() {
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

	input.SignUpEnabled = signUpEnabled
	input.SignUpAutoActivateEnabled = signUpAutoActivateEnabled
	input.TOTPRequired = totpRequired
	input.TOTPSMSEnabled = totpSMSEnabled
	input.MagicLinkSignInEnabled = magicLinkSignInEnabled

	input.GoogleSignInEnabled = googleSignInEnabled

	if input.GoogleSignInClientID, err = NewGoogleClientID(googleSignInClientID); err != nil {
		errs.Set("google sign in client id", err)
	}

	input.FacebookSignInEnabled = facebookSignInEnabled

	if input.FacebookSignInAppID, err = NewFacebookAppID(facebookSignInAppID); err != nil {
		errs.Set("facebook sign in app id", err)
	}
	if input.FacebookSignInAppSecret, err = NewFacebookAppSecret(facebookSignInAppSecret); err != nil {
		errs.Set("facebook sign in app secret", err)
	}

	if input.ResendAPIKey, err = NewResendAPIKey(resendAPIKey); err != nil {
		errs.Set("resend API key", err)
	}
	if input.TwilioSID, err = NewTwilioSID(twilioSID); err != nil {
		errs.Set("twilio sid", err)
	}
	if input.TwilioToken, err = NewTwilioToken(twilioToken); err != nil {
		errs.Set("twilio token", err)
	}
	if input.TwilioFromTel, err = NewTwilioTel(twilioFromTel); err != nil {
		errs.Set("twilio from tel", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) UpdateConfig(ctx context.Context, guard UpdateConfigGuard,
	systemEmail, securityEmail string,
	signUpEnabled, signUpAutoActivateEnabled bool,
	totpRequired, totpSMSEnabled bool,
	magicLinkSignInEnabled bool,
	googleSignInEnabled bool, googleSignInClientID string,
	facebookSignInEnabled bool, facebookSignInAppID, facebookSignInAppSecret string,
	resendAPIKey string,
	twilioSID, twilioToken, twilioFromTel string,
) (*Config, error) {
	input, err := s.UpdateConfigValidate(
		guard,
		systemEmail, securityEmail,
		signUpEnabled, signUpAutoActivateEnabled,
		totpRequired, totpSMSEnabled,
		magicLinkSignInEnabled,
		googleSignInEnabled, googleSignInClientID,
		facebookSignInEnabled, facebookSignInAppID, facebookSignInAppSecret,
		resendAPIKey,
		twilioSID, twilioToken, twilioFromTel,
	)
	if err != nil {
		return nil, err
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("find config: %w", err)
	}

	var errs errsx.Slice

	config.ChangeSystemEmail(input.SystemEmail)
	config.ChangeSecurityEmail(input.SecurityEmail)

	if input.SignUpEnabled {
		config.EnableSignUp()
	} else {
		config.DisableSignUp()
	}

	if input.SignUpAutoActivateEnabled {
		config.EnableSignUpAutoActivate()
	} else {
		config.DisableSignUpAutoActivate()
	}

	if input.TOTPRequired {
		config.EnableTOTPRequired()
	} else {
		config.DisableTOTPRequired()
	}

	if input.MagicLinkSignInEnabled {
		config.EnableMagicLinkSignIn()
	} else {
		config.DisableMagicLinkSignIn()
	}

	config.ChangeGoogleSignInClientID(input.GoogleSignInClientID)

	if input.GoogleSignInEnabled {
		errs.Append(config.EnableGoogleSignIn())
	} else {
		config.DisableGoogleSignIn()
	}

	config.ChangeFacebookSignInAppID(input.FacebookSignInAppID)
	config.ChangeFacebookSignInAppSecret(input.FacebookSignInAppSecret)

	if input.FacebookSignInEnabled {
		errs.Append(config.EnableFacebookSignIn())
	} else {
		config.DisableFacebookSignIn()
	}

	config.ChangeResendAPI(input.ResendAPIKey)
	config.ChangeTwilioAPI(input.TwilioSID, input.TwilioToken, input.TwilioFromTel)

	if input.TOTPSMSEnabled {
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
