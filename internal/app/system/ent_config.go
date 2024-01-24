package system

import (
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type Config struct {
	aggregate.Root

	SystemEmail               string
	SecurityEmail             string
	SignUpEnabled             bool
	SignUpAutoActivateEnabled bool
	TOTPRequired              bool
	TOTPSMSEnabled            bool
	GoogleSignInEnabled       bool
	GoogleSignInClientID      string
	FacebookSignInEnabled     bool
	FacebookSignInAppID       string
	FacebookSignInAppSecret   string
	ResendAPIKey              string
	TwilioSID                 string
	TwilioToken               string
	TwilioFromTel             string
	SetupRequired             bool
}

func (c *Config) HasSMS() bool {
	return c.TwilioSID != "" && c.TwilioToken != "" && c.TwilioFromTel != ""
}

func (c *Config) ChangeSystemEmail(systemEmail Email) {
	c.SystemEmail = systemEmail.String()
}

func (c *Config) ChangeSecurityEmail(securityEmail Email) {
	c.SecurityEmail = securityEmail.String()
}

func (c *Config) EnableSignUp() {
	c.SignUpEnabled = true
}

func (c *Config) DisableSignUp() {
	c.SignUpEnabled = false
}

func (c *Config) EnableSignUpAutoActivate() {
	c.SignUpAutoActivateEnabled = true
}

func (c *Config) DisableSignUpAutoActivate() {
	c.SignUpAutoActivateEnabled = false
}

func (c *Config) EnableTOTPRequired() {
	c.TOTPRequired = true
}

func (c *Config) DisableTOTPRequired() {
	c.TOTPRequired = false
}

func (c *Config) EnableTOTPSMS() error {
	var errs errsx.Map

	if c.TwilioSID == "" {
		errs.Set("twilio sid", "required when two-factor authentication SMS is enabled")
	}
	if c.TwilioToken == "" {
		errs.Set("twilio token", "required when two-factor authentication SMS is enabled")
	}
	if c.TwilioFromTel == "" {
		errs.Set("twilio from tel", "required when two-factor authentication SMS is enabled")
	}

	if errs != nil {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errs)
	}

	c.TOTPSMSEnabled = true

	return nil
}

func (c *Config) DisableTOTPSMS() {
	c.TOTPSMSEnabled = false
}

func (c *Config) ChangeGoogleSignInClientID(clientID GoogleClientID) {
	c.GoogleSignInClientID = clientID.String()
}

func (c *Config) EnableGoogleSignIn() error {
	if c.GoogleSignInClientID == "" {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"google sign in client id": errors.New("required when Google sign in is enabled"),
		})
	}

	c.GoogleSignInEnabled = true

	return nil
}

func (c *Config) DisableGoogleSignIn() {
	c.GoogleSignInEnabled = false
}

func (c *Config) ChangeFacebookSignInAppID(appID FacebookAppID) {
	c.FacebookSignInAppID = appID.String()
}

func (c *Config) ChangeFacebookSignInAppSecret(appSecret FacebookAppSecret) {
	c.FacebookSignInAppSecret = appSecret.String()
}

func (c *Config) EnableFacebookSignIn() error {
	var errs errsx.Map

	if c.FacebookSignInAppID == "" {
		errs.Set("facebook sign in app id", "required when Facebook sign in is enabled")
	}
	if c.FacebookSignInAppSecret == "" {
		errs.Set("facebook sign in app secret", "required when Facebook sign in is enabled")
	}
	if errs != nil {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errs)
	}

	c.FacebookSignInEnabled = true

	return nil
}

func (c *Config) DisableFacebookSignIn() {
	c.FacebookSignInEnabled = false
}

func (c *Config) ChangeResendAPI(apiKey ResendAPIKey) {
	c.ResendAPIKey = apiKey.String()
}

func (c *Config) ChangeTwilioAPI(twilioSID TwilioSID, twilioToken TwilioToken, twilioFromTel TwilioTel) {
	c.TwilioSID = twilioSID.String()
	c.TwilioToken = twilioToken.String()
	c.TwilioFromTel = twilioFromTel.String()
}
