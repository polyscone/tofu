package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/sms"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

var tenants = make(map[string]Tenant)

var databases = struct {
	mu   sync.Mutex
	data map[string]*sql.DB
}{data: make(map[string]*sql.DB)}

func newTenant(hostname string) (*handler.Tenant, error) {
	ctx := context.Background()

	data, ok := tenants[hostname]
	if !ok {
		return nil, errors.Tracef(web.ErrTenantNotFound, "tenant %q not found", hostname)
	}
	if data.Alias == "" {
		return nil, errors.Tracef("alias name for the tenant %q is empty", hostname)
	}

	databases.mu.Lock()

	db := databases.data[data.Alias]
	if db == nil {
		var err error
		p := filepath.Join(opts.data, data.Alias, "main.sqlite")
		db, err = sqlite.Open(ctx, sqlite.KindFile, p)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		databases.data[data.Alias] = db
	}

	databases.mu.Unlock()

	broker := event.NewMemoryBroker()

	accountRepo, err := sqlite.NewAccountRepo(ctx, db)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	webRepo, err := sqlite.NewWebRepo(ctx, db, 2*time.Hour)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	mailer, err := smtp.NewMailClient("localhost", 25)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	messager := sms.NewTwilioClient(&client, data.Twilio.SID, data.Twilio.Token)

	tenant := handler.Tenant{
		Dev:      opts.dev,
		Insecure: opts.server.insecure,
		Proxies:  opts.server.proxies,
		Broker:   broker,
		Email: handler.Email{
			From:   data.Email.From,
			Mailer: mailer,
		},
		SMS: handler.SMS{
			IsConfigured: data.Twilio.SID != "" && data.Twilio.Token != "" && data.Twilio.From != "",
			From:         data.Twilio.From,
			Messager:     messager,
		},
		Account: account.NewService(broker, accountRepo, hasher),
		Repo: handler.Repo{
			Account: accountRepo,
			Web:     webRepo,
		},
	}

	return &tenant, nil
}

type Email struct {
	From string `json:"from"`
}

type Twilio struct {
	SID   string `json:"sid"`
	Token string `json:"token"`
	From  string `json:"from"`
}

type Tenant struct {
	Alias      string   `json:",omitempty"`
	Hostnames  []string `json:"hostnames"`
	Email      Email    `json:"email"`
	Twilio     Twilio   `json:"twilio"`
	IsDisabled bool     `json:"isDisabled"`
}

func initTenants() error {
	tenants = make(map[string]Tenant)
	value := opts.tenants

	f, err := os.OpenFile(value, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return errors.Tracef(err)
	}

	err = func() error {
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return errors.Tracef(err)
		}

		if info.Size() == 0 {
			example := map[string]Tenant{
				"example": {
					Hostnames: []string{"localhost", "local.example.com"},
					Email: Email{
						From: "noreply@example.com",
					},
					Twilio: Twilio{
						SID:   "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
						Token: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
						From:  "+00 0000 000000",
					},
					IsDisabled: true,
				},
			}

			b, err := json.MarshalIndent(example, "", "\t")
			if err != nil {
				return errors.Tracef(err)
			}

			if _, err := f.Write(b); err != nil {
				return errors.Tracef(err)
			}
		}

		return nil
	}()
	if err != nil {
		return errors.Tracef(err)
	}

	if b, err := os.ReadFile(value); err == nil {
		value = string(b)
	}

	data := make(map[string]Tenant)
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return errors.Tracef(err)
	}

	var errs errors.Map
	for alias, tenant := range data {
		if len(tenant.Hostnames) == 0 {
			errs.Set(alias+".hostnames", "must be populated with at least one hostname")
		}
		if tenant.Email.From == "" {
			errs.Set(alias+".email.from", "cannot be empty")
		}

		if alias == "" && len(tenant.Hostnames) != 0 {
			for _, hostname := range tenant.Hostnames {
				errs.Set("hostname "+hostname, "alias cannot be empty")
			}

			continue
		}

		for _, hostname := range tenant.Hostnames {
			if dupe, ok := tenants[hostname]; ok {
				errs.Set(hostname, fmt.Sprintf("cannot associate with %q; already associated with %q", alias, dupe.Alias))
			}

			tenant.Alias = alias

			if !tenant.IsDisabled {
				tenants[hostname] = tenant
			}
		}
	}

	return errs.Tracef("tenant configuration errors")
}
