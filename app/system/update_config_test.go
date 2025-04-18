package system_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type updateConfigGuard struct {
	value bool
}

func (g updateConfigGuard) CanUpdateConfig() bool {
	return g.value
}

func TestUpdateConfig(t *testing.T) {
	validGuard := updateConfigGuard{value: true}
	invalidGuard := updateConfigGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		systemEmail := "foo@example.com"
		securityEmail := "bar@example.com"
		signUpEnabled := false
		signUpAutoActivateEnabled := false
		totpRequired := true
		totpSMSEnabled := true
		magicLinkSignInEnabled := true
		googleSignInEnabled := true
		googleSignInClientID := "1234abcd"
		facebookSignInEnabled := true
		facebookSignInAppID := "abcd1234"
		facebookSignInAppSecret := "fbsecret"
		resendAPIKey := "re_abcdEFG0123456_RE_abcdEFG0123456Z"
		twilioSID := "AC0123456789abcdef0123456789abcdef"
		twilioToken := "0123456789abcdef0123456789abcdef"
		twilioFromTel := "+00 00 0000 0000"

		_, err := svc.UpdateConfig(ctx, validGuard, system.UpdateConfigInput{
			SystemEmail:               systemEmail,
			SecurityEmail:             securityEmail,
			SignUpEnabled:             signUpEnabled,
			SignUpAutoActivateEnabled: signUpAutoActivateEnabled,
			TOTPRequired:              totpRequired,
			TOTPSMSEnabled:            totpSMSEnabled,
			MagicLinkSignInEnabled:    magicLinkSignInEnabled,
			GoogleSignInEnabled:       googleSignInEnabled,
			GoogleSignInClientID:      googleSignInClientID,
			FacebookSignInEnabled:     facebookSignInEnabled,
			FacebookSignInAppID:       facebookSignInAppID,
			FacebookSignInAppSecret:   facebookSignInAppSecret,
			ResendAPIKey:              resendAPIKey,
			TwilioSID:                 twilioSID,
			TwilioToken:               twilioToken,
			TwilioFromTel:             twilioFromTel,
		})
		if err != nil {
			t.Fatal(err)
		}

		config := errsx.Must(repo.FindConfig(ctx))

		if want, got := systemEmail, config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := securityEmail, config.SecurityEmail; want != got {
			t.Errorf("want security email to be %q; got %q", want, got)
		}
		if want, got := signUpEnabled, config.SignUpEnabled; want != got {
			t.Errorf("want sign up enabled to be %v; got %v", want, got)
		}
		if want, got := signUpAutoActivateEnabled, config.SignUpAutoActivateEnabled; want != got {
			t.Errorf("want sign up auto activate enabled to be %v; got %v", want, got)
		}
		if want, got := totpRequired, config.TOTPRequired; want != got {
			t.Errorf("want TOTP required to be %v; got %v", want, got)
		}
		if want, got := totpSMSEnabled, config.TOTPSMSEnabled; want != got {
			t.Errorf("want TOTP SMS enabled to be %v; got %v", want, got)
		}
		if want, got := magicLinkSignInEnabled, config.MagicLinkSignInEnabled; want != got {
			t.Errorf("want magic link sign in enabled to be %v; got %v", want, got)
		}
		if want, got := googleSignInEnabled, config.GoogleSignInEnabled; want != got {
			t.Errorf("want google sign in enabled to be %v; got %v", want, got)
		}
		if want, got := googleSignInClientID, config.GoogleSignInClientID; want != got {
			t.Errorf("want google sign in client id to be %q; got %q", want, got)
		}
		if want, got := facebookSignInEnabled, config.FacebookSignInEnabled; want != got {
			t.Errorf("want facebook sign in enabled to be %v; got %v", want, got)
		}
		if want, got := facebookSignInAppID, config.FacebookSignInAppID; want != got {
			t.Errorf("want facebook sign in app id to be %q; got %q", want, got)
		}
		if want, got := facebookSignInAppSecret, config.FacebookSignInAppSecret; want != got {
			t.Errorf("want facebook sign in app secret to be %q; got %q", want, got)
		}
		if want, got := twilioSID, config.TwilioSID; want != got {
			t.Errorf("want twilio sid to be %q; got %q", want, got)
		}
		if want, got := resendAPIKey, config.ResendAPIKey; want != got {
			t.Errorf("want resend API key to be %q; got %q", want, got)
		}
		if want, got := twilioToken, config.TwilioToken; want != got {
			t.Errorf("want twilio token to be %q; got %q", want, got)
		}
		if want, got := twilioFromTel, config.TwilioFromTel; want != got {
			t.Errorf("want twilio from tel to be %q; got %q", want, got)
		}

		systemEmail = "bar@example.com"
		securityEmail = "baz@example.com"
		signUpEnabled = true
		signUpAutoActivateEnabled = true
		totpRequired = false
		totpSMSEnabled = false
		magicLinkSignInEnabled = false
		googleSignInEnabled = false
		googleSignInClientID = "xyz"
		facebookSignInEnabled = false
		facebookSignInAppID = "xxx"
		facebookSignInAppSecret = "zzz"
		resendAPIKey = ""
		twilioSID = ""
		twilioToken = ""
		twilioFromTel = ""

		_, err = svc.UpdateConfig(ctx, validGuard, system.UpdateConfigInput{
			SystemEmail:               systemEmail,
			SecurityEmail:             securityEmail,
			SignUpEnabled:             signUpEnabled,
			SignUpAutoActivateEnabled: signUpAutoActivateEnabled,
			TOTPRequired:              totpRequired,
			TOTPSMSEnabled:            totpSMSEnabled,
			MagicLinkSignInEnabled:    magicLinkSignInEnabled,
			GoogleSignInEnabled:       googleSignInEnabled,
			GoogleSignInClientID:      googleSignInClientID,
			FacebookSignInEnabled:     facebookSignInEnabled,
			FacebookSignInAppID:       facebookSignInAppID,
			FacebookSignInAppSecret:   facebookSignInAppSecret,
			ResendAPIKey:              resendAPIKey,
			TwilioSID:                 twilioSID,
			TwilioToken:               twilioToken,
			TwilioFromTel:             twilioFromTel,
		})
		if err != nil {
			t.Fatal(err)
		}

		config = errsx.Must(repo.FindConfig(ctx))

		if want, got := systemEmail, config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := securityEmail, config.SecurityEmail; want != got {
			t.Errorf("want security email to be %q; got %q", want, got)
		}
		if want, got := signUpEnabled, config.SignUpEnabled; want != got {
			t.Errorf("want sign up enabled to be %v; got %v", want, got)
		}
		if want, got := signUpAutoActivateEnabled, config.SignUpAutoActivateEnabled; want != got {
			t.Errorf("want sign up auto activate enabled to be %v; got %v", want, got)
		}
		if want, got := totpRequired, config.TOTPRequired; want != got {
			t.Errorf("want TOTP required to be %v; got %v", want, got)
		}
		if want, got := totpSMSEnabled, config.TOTPSMSEnabled; want != got {
			t.Errorf("want TOTP SMS enabled to be %v; got %v", want, got)
		}
		if want, got := magicLinkSignInEnabled, config.MagicLinkSignInEnabled; want != got {
			t.Errorf("want magic link sign in enabled to be %v; got %v", want, got)
		}
		if want, got := googleSignInEnabled, config.GoogleSignInEnabled; want != got {
			t.Errorf("want google sign in enabled to be %v; got %v", want, got)
		}
		if want, got := googleSignInClientID, config.GoogleSignInClientID; want != got {
			t.Errorf("want google sign in client id to be %q; got %q", want, got)
		}
		if want, got := facebookSignInEnabled, config.FacebookSignInEnabled; want != got {
			t.Errorf("want facebook sign in enabled to be %v; got %v", want, got)
		}
		if want, got := facebookSignInAppID, config.FacebookSignInAppID; want != got {
			t.Errorf("want facebook sign in app id to be %q; got %q", want, got)
		}
		if want, got := facebookSignInAppSecret, config.FacebookSignInAppSecret; want != got {
			t.Errorf("want facebook sign in app secret to be %q; got %q", want, got)
		}
		if want, got := resendAPIKey, config.ResendAPIKey; want != got {
			t.Errorf("want resend API key to be %q; got %q", want, got)
		}
		if want, got := twilioSID, config.TwilioSID; want != got {
			t.Errorf("want twilio sid to be %q; got %q", want, got)
		}
		if want, got := twilioToken, config.TwilioToken; want != got {
			t.Errorf("want twilio token to be %q; got %q", want, got)
		}
		if want, got := twilioFromTel, config.TwilioFromTel; want != got {
			t.Errorf("want twilio from tel to be %q; got %q", want, got)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		type vals map[string]any
		c := func(overrides vals) system.Config {
			config := system.Config{
				SystemEmail:   "a@a.com",
				SecurityEmail: "b@b.com",
				ResendAPIKey:  "re_abcdEFG0123456_RE_abcdEFG0123456Z",
				TwilioSID:     "AC0123456789abcdef0123456789abcdef",
				TwilioToken:   "0123456789abcdef0123456789abcdef",
				TwilioFromTel: "+00 00 0000 0000",
			}

			for key, value := range overrides {
				switch key {
				case "SystemEmail":
					config.SystemEmail = value.(string)

				case "SecurityEmail":
					config.SecurityEmail = value.(string)

				case "TOTPSMSEnabled":
					config.TOTPSMSEnabled = value.(bool)

				case "GoogleSignInEnabled":
					config.GoogleSignInEnabled = value.(bool)

				case "FacebookSignInEnabled":
					config.FacebookSignInEnabled = value.(bool)

				case "FacebookSignInAppID":
					config.FacebookSignInAppID = value.(string)

				case "FacebookSignInAppSecret":
					config.FacebookSignInAppSecret = value.(string)

				case "ResendAPIKey":
					config.ResendAPIKey = value.(string)

				case "TwilioSID":
					config.TwilioSID = value.(string)

				case "TwilioToken":
					config.TwilioToken = value.(string)

				case "TwilioFromTel":
					config.TwilioFromTel = value.(string)

				default:
					panic(fmt.Sprintf("unknown test key %q", key))
				}
			}

			return config
		}

		tt := []struct {
			name      string
			guard     updateConfigGuard
			overrides vals
			want      error
		}{
			{"invalid guard", invalidGuard, nil, app.ErrForbidden},
			{"empty system email", validGuard, vals{"SystemEmail": ""}, app.ErrMalformedInput},
			{"malformed system email", validGuard, vals{"SystemEmail": "a"}, app.ErrMalformedInput},
			{"empty security email", validGuard, vals{"SecurityEmail": ""}, app.ErrMalformedInput},
			{"malformed security email", validGuard, vals{"SecurityEmail": "a"}, app.ErrMalformedInput},
			{"malformed resend API key", validGuard, vals{"ResendAPIKey": "a"}, app.ErrMalformedInput},
			{"TOTP SMS enabled without Twilio SID", validGuard, vals{"TOTPSMSEnabled": true, "TwilioSID": ""}, app.ErrInvalidInput},
			{"TOTP SMS enabled without Twilio token", validGuard, vals{"TOTPSMSEnabled": true, "TwilioToken": ""}, app.ErrInvalidInput},
			{"TOTP SMS enabled without Twilio from tel", validGuard, vals{"TOTPSMSEnabled": true, "TwilioFromTel": ""}, app.ErrInvalidInput},
			{"google sign in enabled without client id", validGuard, vals{"GoogleSignInEnabled": true}, app.ErrInvalidInput},
			{"facebook sign in enabled without app id", validGuard, vals{"FacebookSignInEnabled": true, "FacebookSignInAppSecret": "123"}, app.ErrInvalidInput},
			{"facebook sign in enabled without app secret", validGuard, vals{"FacebookSignInEnabled": true, "FacebookSignInAppID": "123"}, app.ErrInvalidInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				config := c(tc.overrides)

				_, err := svc.UpdateConfig(ctx, tc.guard, system.UpdateConfigInput{
					SystemEmail:               config.SystemEmail,
					SecurityEmail:             config.SecurityEmail,
					SignUpEnabled:             config.SignUpEnabled,
					SignUpAutoActivateEnabled: config.SignUpAutoActivateEnabled,
					TOTPRequired:              config.TOTPRequired,
					TOTPSMSEnabled:            config.TOTPSMSEnabled,
					MagicLinkSignInEnabled:    config.MagicLinkSignInEnabled,
					GoogleSignInEnabled:       config.GoogleSignInEnabled,
					GoogleSignInClientID:      config.GoogleSignInClientID,
					FacebookSignInEnabled:     config.FacebookSignInEnabled,
					FacebookSignInAppID:       config.FacebookSignInAppID,
					FacebookSignInAppSecret:   config.FacebookSignInAppSecret,
					ResendAPIKey:              config.ResendAPIKey,
					TwilioSID:                 config.TwilioSID,
					TwilioToken:               config.TwilioToken,
					TwilioFromTel:             config.TwilioFromTel,
				})
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
