package system

import "github.com/polyscone/tofu/internal/pkg/aggregate"

type Config struct {
	aggregate.Root

	SystemEmail   string
	TwilioSID     string
	TwilioToken   string
	TwilioFromTel string
	IsSetup       bool
}

func (c *Config) HasSMS() bool {
	return c.TwilioSID != "" && c.TwilioToken != "" && c.TwilioFromTel != ""
}

func (c *Config) ChangeSystemEmail(systemEmail Email) {
	c.SystemEmail = systemEmail.String()
}

func (c *Config) ChangeTwilioAPI(twilioSID TwilioSID, twilioToken TwilioToken, twilioFromTel TwilioTel) {
	c.TwilioSID = twilioSID.String()
	c.TwilioToken = twilioToken.String()
	c.TwilioFromTel = twilioFromTel.String()
}
