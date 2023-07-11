package system_test

import (
	"context"
	"errors"
	"fmt"
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
		securityEmail := "bar@example.com"
		requireTOTP := true
		googleSignInEnabled := true
		googleSignInClientID := "1234abcd"
		twilioSID := ""
		twilioToken := ""
		twilioFromTel := ""

		_, err := svc.UpdateConfig(ctx, validGuard,
			systemEmail,
			securityEmail,
			requireTOTP,
			googleSignInEnabled,
			googleSignInClientID,
			twilioSID,
			twilioToken,
			twilioFromTel,
		)
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
		if want, got := requireTOTP, config.RequireTOTP; want != got {
			t.Errorf("want require TOTP to be %v; got %v", want, got)
		}
		if want, got := googleSignInEnabled, config.GoogleSignInEnabled; want != got {
			t.Errorf("want google sign in enabled to be %v; got %v", want, got)
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
		securityEmail = "baz@example.com"
		requireTOTP = false
		googleSignInEnabled = false
		googleSignInClientID = "xyz"
		twilioSID = "AC0123456789abcdef0123456789abcdef"
		twilioToken = "0123456789abcdef0123456789abcdef"
		twilioFromTel = "+00 00 0000 0000"

		_, err = svc.UpdateConfig(ctx, validGuard,
			systemEmail,
			securityEmail,
			requireTOTP,
			googleSignInEnabled,
			googleSignInClientID,
			twilioSID,
			twilioToken,
			twilioFromTel,
		)
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
		if want, got := requireTOTP, config.RequireTOTP; want != got {
			t.Errorf("want require TOTP to be %v; got %v", want, got)
		}
		if want, got := googleSignInEnabled, config.GoogleSignInEnabled; want != got {
			t.Errorf("want google sign in enabled to be %v; got %v", want, got)
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

		type vals map[string]any
		c := func(overrides vals) system.Config {
			config := system.Config{
				SystemEmail:   "a@a.com",
				SecurityEmail: "b@b.com",
			}

			for key, value := range overrides {
				switch key {
				case "SystemEmail":
					config.SystemEmail = value.(string)

				case "SecurityEmail":
					config.SecurityEmail = value.(string)

				case "GoogleSignInEnabled":
					config.GoogleSignInEnabled = value.(bool)

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
			{"unauthorised", invalidGuard, nil, app.ErrUnauthorised},
			{"empty system email", validGuard, vals{"SystemEmail": ""}, app.ErrMalformedInput},
			{"malformed system email", validGuard, vals{"SystemEmail": "a"}, app.ErrMalformedInput},
			{"empty security email", validGuard, vals{"SecurityEmail": ""}, app.ErrMalformedInput},
			{"malformed security email", validGuard, vals{"SecurityEmail": "a"}, app.ErrMalformedInput},
			{"google sign in enabled without client id", validGuard, vals{"GoogleSignInEnabled": true}, app.ErrInvalidInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				config := c(tc.overrides)

				_, err := svc.UpdateConfig(ctx, tc.guard,
					config.SystemEmail,
					config.SecurityEmail,
					config.RequireTOTP,
					config.GoogleSignInEnabled,
					config.GoogleSignInClientID,
					config.TwilioSID,
					config.TwilioToken,
					config.TwilioFromTel,
				)
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
