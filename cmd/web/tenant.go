package main

import (
	"context"
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
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/sms"
)

var databases = struct {
	mu   sync.Mutex
	data map[string]*sqlite.DB
}{data: make(map[string]*sqlite.DB)}

func newTenant(hostname string) (*handler.Tenant, error) {
	ctx := context.Background()

	data := opts.tenants.lookup[hostname]
	if data.Common == "" {
		return nil, errors.Tracef("common name for the tenant %q is empty", hostname)
	}

	databases.mu.Lock()

	db := databases.data[data.Common]
	if db == nil {
		var err error
		p := filepath.Join(opts.data, data.Common, "main.db")
		db, err = sqlite.Open(ctx, sqlite.KindFile, p)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		databases.data[data.Common] = db
	}

	databases.mu.Unlock()

	bus, broker, err := app.Compose(ctx, db, []byte(opts.secret), hasher)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	sessions, err := web.NewSQLiteSessionRepo(ctx, db, 2*time.Hour)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	tokens, err := web.NewSQLiteTokenRepo(ctx, db)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	mailer, err := smtp.NewMailClient("localhost", 25)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	messager := sms.NewTwilioClient(&client, data.Twilio.SID, data.Twilio.Token)

	tenant := &handler.Tenant{
		Dev:      opts.dev,
		Insecure: opts.server.insecure,
		Proxies:  opts.server.proxies,
		Bus:      bus,
		Broker:   broker,
		Sessions: sessions,
		Tokens:   tokens,
		Email:    mailer,
		SMS:      messager,
		SMSFrom:  data.Twilio.From,
	}

	return tenant, nil
}

type twilio struct {
	SID   string
	Token string
	From  string
}

type tenant struct {
	Common    string
	Hostnames []string
	Twilio    twilio
}

type tenants struct {
	data   map[string]tenant
	lookup map[string]tenant
}

func (t *tenants) Set(value string) error {
	if value == "" {
		return nil
	}

	if b, err := os.ReadFile(value); err == nil {
		value = string(b)
	}

	if err := json.Unmarshal([]byte(value), &t.data); err != nil {
		return errors.Tracef(err)
	}

	if t.lookup == nil {
		t.lookup = make(map[string]tenant)
	}

	var errs errors.Map
	for common, tenant := range t.data {
		for _, hostname := range tenant.Hostnames {
			if dupe, ok := t.lookup[hostname]; ok {
				errs.Set(hostname, fmt.Sprintf("cannot associate with %q, already associated with %q", dupe, common))
			}

			tenant.Common = common

			t.lookup[hostname] = tenant
		}
	}

	return errs.Tracef("duplicate tenant hostnames")
}

func (t tenants) String() string {
	if t.data == nil {
		return ""
	}

	b, err := json.Marshal(t.data)
	if err != nil {
		panic(err)
	}

	return string(b)
}
