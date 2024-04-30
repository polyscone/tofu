package system

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type UpdateConfigGuard interface {
	CanUpdateConfig() bool
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
) error {
	var input struct {
		systemEmail               Email
		securityEmail             Email
		signUpEnabled             bool
		signUpAutoActivateEnabled bool
		totpRequired              bool
		totpSMSEnabled            bool
		magicLinkSignInEnabled    bool
		googleSignInEnabled       bool
		googleSignInClientID      GoogleClientID
		facebookSignInEnabled     bool
		facebookSignInAppID       FacebookAppID
		facebookSignInAppSecret   FacebookAppSecret
		resendAPIKey              ResendAPIKey
		twilioSID                 TwilioSID
		twilioToken               TwilioToken
		twilioFromTel             TwilioTel
	}
	{
		if !guard.CanUpdateConfig() {
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

		input.signUpEnabled = signUpEnabled
		input.signUpAutoActivateEnabled = signUpAutoActivateEnabled
		input.totpRequired = totpRequired
		input.totpSMSEnabled = totpSMSEnabled
		input.magicLinkSignInEnabled = magicLinkSignInEnabled

		input.googleSignInEnabled = googleSignInEnabled

		if input.googleSignInClientID, err = NewGoogleClientID(googleSignInClientID); err != nil {
			errs.Set("google sign in client id", err)
		}

		input.facebookSignInEnabled = facebookSignInEnabled

		if input.facebookSignInAppID, err = NewFacebookAppID(facebookSignInAppID); err != nil {
			errs.Set("facebook sign in app id", err)
		}
		if input.facebookSignInAppSecret, err = NewFacebookAppSecret(facebookSignInAppSecret); err != nil {
			errs.Set("facebook sign in app secret", err)
		}

		if input.resendAPIKey, err = NewResendAPIKey(resendAPIKey); err != nil {
			errs.Set("resend API key", err)
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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	config, err := s.repo.FindConfig(ctx)
	if err != nil {
		return fmt.Errorf("find config: %w", err)
	}

	var errs errsx.Slice

	config.ChangeSystemEmail(input.systemEmail)
	config.ChangeSecurityEmail(input.securityEmail)

	if input.signUpEnabled {
		config.EnableSignUp()
	} else {
		config.DisableSignUp()
	}

	if input.signUpAutoActivateEnabled {
		config.EnableSignUpAutoActivate()
	} else {
		config.DisableSignUpAutoActivate()
	}

	if input.totpRequired {
		config.EnableTOTPRequired()
	} else {
		config.DisableTOTPRequired()
	}

	if input.magicLinkSignInEnabled {
		config.EnableMagicLinkSignIn()
	} else {
		config.DisableMagicLinkSignIn()
	}

	config.ChangeGoogleSignInClientID(input.googleSignInClientID)

	if input.googleSignInEnabled {
		errs.Append(config.EnableGoogleSignIn())
	} else {
		config.DisableGoogleSignIn()
	}

	config.ChangeFacebookSignInAppID(input.facebookSignInAppID)
	config.ChangeFacebookSignInAppSecret(input.facebookSignInAppSecret)

	if input.facebookSignInEnabled {
		errs.Append(config.EnableFacebookSignIn())
	} else {
		config.DisableFacebookSignIn()
	}

	config.ChangeResendAPI(input.resendAPIKey)
	config.ChangeTwilioAPI(input.twilioSID, input.twilioToken, input.twilioFromTel)

	if input.totpSMSEnabled {
		errs.Append(config.EnableTOTPSMS())
	} else {
		config.DisableTOTPSMS()
	}

	if errs != nil {
		return errs
	}

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	s.broker.Flush(&config.Events)

	return nil
}
