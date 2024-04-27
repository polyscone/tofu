package system

import (
	"context"
	"testing"
)

type Reader interface {
	FindConfig(ctx context.Context) (*Config, error)
}

type Writer interface {
	SaveConfig(ctx context.Context, config *Config) error
}

type ReadWriter interface {
	Reader
	Writer
}

func TestRepo(ctx context.Context, t *testing.T, newRepo func() ReadWriter) {
	t.Run("config", func(t *testing.T) { testRepoConfig(ctx, t, newRepo) })
}

func testRepoConfig(ctx context.Context, t *testing.T, newRepo func() ReadWriter) {
	t.Run("find and save", func(t *testing.T) {
		repo := newRepo()

		config, err := repo.FindConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := true, config.SetupRequired; want != got {
			t.Errorf("want setup required to be %v; got %v", want, got)
		}
		if want, got := "", config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := "", config.SecurityEmail; want != got {
			t.Errorf("want security email to be %q; got %q", want, got)
		}
		if want, got := true, config.SignUpEnabled; want != got {
			t.Errorf("want sign up enabled to be %v; got %v", want, got)
		}
		if want, got := true, config.SignUpAutoActivateEnabled; want != got {
			t.Errorf("want sign up auto activate enabled to be %v; got %v", want, got)
		}
		if want, got := false, config.TOTPRequired; want != got {
			t.Errorf("want TOTP required to be %v; got %v", want, got)
		}
		if want, got := false, config.TOTPSMSEnabled; want != got {
			t.Errorf("want TOTP SMS enabled to be %v; got %v", want, got)
		}
		if want, got := false, config.MagicLinkSignInEnabled; want != got {
			t.Errorf("want magic link sign in enabled to be %v; got %v", want, got)
		}
		if want, got := false, config.GoogleSignInEnabled; want != got {
			t.Errorf("want Google sign in enabled to be %v; got %v", want, got)
		}
		if want, got := "", config.GoogleSignInClientID; want != got {
			t.Errorf("want Google sign in client id to be %q; got %q", want, got)
		}
		if want, got := false, config.FacebookSignInEnabled; want != got {
			t.Errorf("want Facebook sign in enabled to be %v; got %v", want, got)
		}
		if want, got := "", config.FacebookSignInAppID; want != got {
			t.Errorf("want Facebook sign in app id to be %q; got %q", want, got)
		}
		if want, got := "", config.FacebookSignInAppSecret; want != got {
			t.Errorf("want Facebook sign in app secret to be %q; got %q", want, got)
		}
		if want, got := "", config.ResendAPIKey; want != got {
			t.Errorf("want resend API key to be %q; got %q", want, got)
		}
		if want, got := "", config.TwilioSID; want != got {
			t.Errorf("want twilio sid to be %q; got %q", want, got)
		}
		if want, got := "", config.TwilioToken; want != got {
			t.Errorf("want twilio token to be %q; got %q", want, got)
		}
		if want, got := "", config.TwilioFromTel; want != got {
			t.Errorf("want twilio from tel to be %q; got %q", want, got)
		}

		err = repo.SaveConfig(ctx, &Config{
			SetupRequired:             true,
			SystemEmail:               "1",
			SecurityEmail:             "2",
			SignUpEnabled:             false,
			SignUpAutoActivateEnabled: false,
			TOTPRequired:              true,
			TOTPSMSEnabled:            true,
			MagicLinkSignInEnabled:    true,
			GoogleSignInEnabled:       true,
			GoogleSignInClientID:      "3",
			FacebookSignInEnabled:     true,
			FacebookSignInAppID:       "4",
			FacebookSignInAppSecret:   "5",
			ResendAPIKey:              "6",
			TwilioSID:                 "7",
			TwilioToken:               "8",
			TwilioFromTel:             "9",
		})
		if err != nil {
			t.Fatal(err)
		}

		config, err = repo.FindConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := false, config.SetupRequired; want != got {
			t.Errorf("want setup required to be %v; got %v", want, got)
		}
		if want, got := "1", config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := "2", config.SecurityEmail; want != got {
			t.Errorf("want security email to be %q; got %q", want, got)
		}
		if want, got := false, config.SignUpEnabled; want != got {
			t.Errorf("want sign up enabled to be %v; got %v", want, got)
		}
		if want, got := false, config.SignUpAutoActivateEnabled; want != got {
			t.Errorf("want sign up auto activate enabled to be %v; got %v", want, got)
		}
		if want, got := true, config.TOTPRequired; want != got {
			t.Errorf("want TOTP required to be %v; got %v", want, got)
		}
		if want, got := true, config.TOTPSMSEnabled; want != got {
			t.Errorf("want TOTP SMS enabled to be %v; got %v", want, got)
		}
		if want, got := true, config.MagicLinkSignInEnabled; want != got {
			t.Errorf("want magic link sign in enabled to be %v; got %v", want, got)
		}
		if want, got := true, config.GoogleSignInEnabled; want != got {
			t.Errorf("want Google sign in enabled to be %v; got %v", want, got)
		}
		if want, got := "3", config.GoogleSignInClientID; want != got {
			t.Errorf("want Google sign in client id to be %q; got %q", want, got)
		}
		if want, got := true, config.FacebookSignInEnabled; want != got {
			t.Errorf("want Facebook sign in enabled to be %v; got %v", want, got)
		}
		if want, got := "4", config.FacebookSignInAppID; want != got {
			t.Errorf("want Facebook sign in app id to be %q; got %q", want, got)
		}
		if want, got := "5", config.FacebookSignInAppSecret; want != got {
			t.Errorf("want Facebook sign in app secret to be %q; got %q", want, got)
		}
		if want, got := "6", config.ResendAPIKey; want != got {
			t.Errorf("want resend API key to be %q; got %q", want, got)
		}
		if want, got := "7", config.TwilioSID; want != got {
			t.Errorf("want twilio sid to be %q; got %q", want, got)
		}
		if want, got := "8", config.TwilioToken; want != got {
			t.Errorf("want twilio token to be %q; got %q", want, got)
		}
		if want, got := "9", config.TwilioFromTel; want != got {
			t.Errorf("want twilio from tel to be %q; got %q", want, got)
		}
	})
}
