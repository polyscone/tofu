package repotest

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/system"
)

func SystemConfig(ctx context.Context, t *testing.T, newRepo func() system.ReadWriter) {
	t.Run("find and save", func(t *testing.T) {
		repo := newRepo()

		config, err := repo.FindConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := true, config.RequireSetup; want != got {
			t.Errorf("want require setup to be %v; got %v", want, got)
		}
		if want, got := "", config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := "", config.SecurityEmail; want != got {
			t.Errorf("want security email to be %q; got %q", want, got)
		}
		if want, got := false, config.RequireTOTP; want != got {
			t.Errorf("want require TOTP to be %v; got %v", want, got)
		}
		if want, got := false, config.GoogleSignInEnabled; want != got {
			t.Errorf("want Google sign in enabled to be %v; got %v", want, got)
		}
		if want, got := "", config.GoogleSignInClientID; want != got {
			t.Errorf("want Google sign in client id to be %q; got %q", want, got)
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

		err = repo.SaveConfig(ctx, &system.Config{
			RequireSetup:         true,
			SystemEmail:          "1",
			SecurityEmail:        "2",
			RequireTOTP:          true,
			GoogleSignInEnabled:  true,
			GoogleSignInClientID: "3",
			TwilioSID:            "4",
			TwilioToken:          "5",
			TwilioFromTel:        "6",
		})
		if err != nil {
			t.Fatal(err)
		}

		config, err = repo.FindConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := false, config.RequireSetup; want != got {
			t.Errorf("want require setup to be %v; got %v", want, got)
		}
		if want, got := "1", config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := "2", config.SecurityEmail; want != got {
			t.Errorf("want security email to be %q; got %q", want, got)
		}
		if want, got := true, config.RequireTOTP; want != got {
			t.Errorf("want require TOTP to be %v; got %v", want, got)
		}
		if want, got := true, config.GoogleSignInEnabled; want != got {
			t.Errorf("want Google sign in enabled to be %v; got %v", want, got)
		}
		if want, got := "3", config.GoogleSignInClientID; want != got {
			t.Errorf("want Google sign in client id to be %q; got %q", want, got)
		}
		if want, got := "4", config.TwilioSID; want != got {
			t.Errorf("want twilio sid to be %q; got %q", want, got)
		}
		if want, got := "5", config.TwilioToken; want != got {
			t.Errorf("want twilio token to be %q; got %q", want, got)
		}
		if want, got := "6", config.TwilioFromTel; want != got {
			t.Errorf("want twilio from tel to be %q; got %q", want, got)
		}
	})
}
