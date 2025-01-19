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

type updateEmailsGuard struct {
	value bool
}

func (g updateEmailsGuard) CanUpdateEmails() bool {
	return g.value
}

func TestUpdateEmails(t *testing.T) {
	validGuard := updateEmailsGuard{value: true}
	invalidGuard := updateEmailsGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		systemEmail := "foo@example.com"
		securityEmail := "bar@example.com"

		_, err := svc.UpdateEmails(ctx, validGuard, system.UpdateEmailsInput{
			SystemEmail:   systemEmail,
			SecurityEmail: securityEmail,
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

		systemEmail = "bar@example.com"
		securityEmail = "baz@example.com"

		_, err = svc.UpdateEmails(ctx, validGuard, system.UpdateEmailsInput{
			SystemEmail:   systemEmail,
			SecurityEmail: securityEmail,
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
			}

			for key, value := range overrides {
				switch key {
				case "SystemEmail":
					config.SystemEmail = value.(string)

				case "SecurityEmail":
					config.SecurityEmail = value.(string)

				default:
					panic(fmt.Sprintf("unknown test key %q", key))
				}
			}

			return config
		}

		tt := []struct {
			name      string
			guard     updateEmailsGuard
			overrides vals
			want      error
		}{
			{"invalid guard", invalidGuard, nil, app.ErrForbidden},
			{"empty system email", validGuard, vals{"SystemEmail": ""}, app.ErrMalformedInput},
			{"malformed system email", validGuard, vals{"SystemEmail": "a"}, app.ErrMalformedInput},
			{"empty security email", validGuard, vals{"SecurityEmail": ""}, app.ErrMalformedInput},
			{"malformed security email", validGuard, vals{"SecurityEmail": "a"}, app.ErrMalformedInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				config := c(tc.overrides)

				_, err := svc.UpdateEmails(ctx, tc.guard, system.UpdateEmailsInput{
					SystemEmail:   config.SystemEmail,
					SecurityEmail: config.SecurityEmail,
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
