package system_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
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

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		systemEmail := "foo@example.com"
		googleSignInClientID := "1234abcd"
		twilioSID := ""
		twilioToken := ""
		twilioFromTel := ""

		_, err := svc.UpdateConfig(ctx, validGuard, systemEmail, googleSignInClientID, twilioSID, twilioToken, twilioFromTel)
		if err != nil {
			t.Fatal(err)
		}

		config := errsx.Must(repo.FindConfig(ctx))

		if want, got := systemEmail, config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := googleSignInClientID, config.GoogleSignInClientID; want != got {
			t.Errorf("want google sign in client id to be %q; got %q", want, got)
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

		systemEmail = "bar@example.com"
		googleSignInClientID = "xyz"
		twilioSID = "AC0123456789abcdef0123456789abcdef"
		twilioToken = "0123456789abcdef0123456789abcdef"
		twilioFromTel = "+00 00 0000 0000"

		_, err = svc.UpdateConfig(ctx, validGuard, systemEmail, googleSignInClientID, twilioSID, twilioToken, twilioFromTel)
		if err != nil {
			t.Fatal(err)
		}

		config = errsx.Must(repo.FindConfig(ctx))

		if want, got := systemEmail, config.SystemEmail; want != got {
			t.Errorf("want system email to be %q; got %q", want, got)
		}
		if want, got := googleSignInClientID, config.GoogleSignInClientID; want != got {
			t.Errorf("want google sign in client id to be %q; got %q", want, got)
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

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  updateConfigGuard
			config system.Config
			want   error
		}{
			{"unauthorised", invalidGuard, system.Config{}, app.ErrUnauthorised},
			{"empty email", validGuard, system.Config{}, app.ErrMalformedInput},
			{"malformed email", validGuard, system.Config{SystemEmail: "a"}, app.ErrMalformedInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.UpdateConfig(ctx, tc.guard,
					tc.config.SystemEmail,
					tc.config.GoogleSignInClientID,
					tc.config.TwilioSID,
					tc.config.TwilioToken,
					tc.config.TwilioFromTel,
				)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
