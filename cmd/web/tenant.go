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

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/sms"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

var tenants = make(map[string]tenant)

var databases = struct {
	mu   sync.Mutex
	data map[string]*sql.DB
}{data: make(map[string]*sql.DB)}

func newTenant(hostname string) (*handler.Tenant, error) {
	ctx := context.Background()

	data := tenants[hostname]
	if data.Alias == "" {
		return nil, errors.Tracef("alias name for the tenant %q is empty", hostname)
	}

	databases.mu.Lock()

	db := databases.data[data.Alias]
	if db == nil {
		var err error
		p := filepath.Join(opts.data, data.Alias, "main.db")
		db, err = sqlite.Open(ctx, sqlite.KindFile, p)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		databases.data[data.Alias] = db
	}

	databases.mu.Unlock()

	broker := event.NewMemoryBroker()

	accountRepo, err := sqlite.NewAccountRepo(ctx, db, []byte(opts.secret))
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
		Email:    mailer,
		SMS:      messager,
		SMSFrom:  data.Twilio.From,

		Account: account.NewService(broker, accountRepo, hasher),

		Repo: handler.Repo{
			Account: accountRepo,
			Web:     webRepo,
		},
	}

	return &tenant, nil
}

type twilio struct {
	SID   string `json:"sid"`
	Token string `json:"token"`
	From  string `json:"from"`
}

type tenant struct {
	Alias     string   `json:",omitempty"`
	Hostnames []string `json:"hostnames"`
	Twilio    twilio   `json:"twilio"`
}

func initTenants() error {
	tenants = make(map[string]tenant)
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
			example := map[string]tenant{
				"example": {
					Hostnames: []string{"localhost"},
					Twilio: twilio{
						SID:   "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
						Token: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
						From:  "+00 0000 000000",
					},
				},
			}

			b, err := json.MarshalIndent(example, "", "  ")
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

	data := make(map[string]tenant)
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return errors.Tracef(err)
	}

	var errs errors.Map
	for alias, tenant := range data {
		for _, hostname := range tenant.Hostnames {
			if dupe, ok := tenants[hostname]; ok {
				errs.Set(hostname, fmt.Sprintf("cannot associate with %q; already associated with %q", dupe, alias))
			}

			tenant.Alias = alias

			tenants[hostname] = tenant
		}
	}

	return errs.Tracef("duplicate tenant hostnames")
}
