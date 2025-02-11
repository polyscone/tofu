package system

import (
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/aggregate"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/i18n"
)

type Config struct {
	aggregate.Root

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
	SetupRequired             bool
}

func (c *Config) WithoutSecrets() *Config {
	config := *c

	config.FacebookSignInAppSecret = ""

	return &config
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
		errs.Set("twilio sid", i18n.M("enable_totp_sms:system.config.error.twilio_sid_required"))
	}
	if c.TwilioToken == "" {
		errs.Set("twilio token", i18n.M("enable_totp_sms:system.config.error.twilio_token_required"))
	}
	if c.TwilioFromTel == "" {
		errs.Set("twilio token", i18n.M("enable_totp_sms:system.config.error.twilio_from_tel_required"))
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

func (c *Config) EnableMagicLinkSignIn() {
	c.MagicLinkSignInEnabled = true
}

func (c *Config) DisableMagicLinkSignIn() {
	c.MagicLinkSignInEnabled = false
}

func (c *Config) EnableGoogleSignIn() error {
	if c.GoogleSignInClientID == "" {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"google sign in client id": i18n.M("enable_google_sign_in:system.config.error.google_sign_in_client_id_required"),
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
		errs.Set("facebook sign in app id", i18n.M("enable_facebook_sign_in:system.config.error.facebook_sign_in_app_id_required"))
	}
	if c.FacebookSignInAppSecret == "" {
		errs.Set("facebook sign in app secret", i18n.M("enable_facebook_sign_in:system.config.error.facebook_sign_in_app_secret_required"))
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
