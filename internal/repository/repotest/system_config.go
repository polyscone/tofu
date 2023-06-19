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

		if want, got := true, config.RequiresSetup; want != got {
			t.Errorf("want require setup to be %v; got %v", want, got)
		}

		err = repo.SaveConfig(ctx, &system.Config{
			SystemEmail:          "1",
			GoogleSignInClientID: "2",
			TwilioSID:            "3",
			TwilioToken:          "4",
			TwilioFromTel:        "5",
		})
		if err != nil {
			t.Fatal(err)
		}

		config, err = repo.FindConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := false, config.RequiresSetup; want != got {
			t.Errorf("want require setup to be %v; got %v", want, got)
		}
		if want, got := "1", config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := "2", config.GoogleSignInClientID; want != got {
			t.Errorf("want Google sign in client id to be %q; got %q", want, got)
		}
		if want, got := "3", config.TwilioSID; want != got {
			t.Errorf("want twilio sid to be %q; got %q", want, got)
		}
		if want, got := "4", config.TwilioToken; want != got {
			t.Errorf("want twilio token to be %q; got %q", want, got)
		}
		if want, got := "5", config.TwilioFromTel; want != got {
			t.Errorf("want twilio from tel to be %q; got %q", want, got)
		}
	})
}
