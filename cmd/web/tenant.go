package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

var databases = struct {
	mu   sync.Mutex
	data map[string]*sqlite.DB
}{data: make(map[string]*sqlite.DB)}

func newTenant(hostname string) (*handler.Tenant, error) {
	ctx := context.Background()

	common := opts.tenants[hostname]
	if common == "" {
		return nil, errors.Tracef("common name for the tenant %q is empty", hostname)
	}

	databases.mu.Lock()

	db := databases.data[common]
	if db == nil {
		var err error
		p := filepath.Join(opts.data, common, "main.db")
		db, err = sqlite.Open(ctx, sqlite.KindFile, p)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		databases.data[common] = db
	}

	databases.mu.Unlock()

	bus, broker, err := app.Compose(ctx, db, []byte(opts.secret))
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

	proxies := strings.Fields(opts.server.proxies)

	tenant := &handler.Tenant{
		Dev:      opts.dev,
		Insecure: opts.server.insecure,
		Proxies:  proxies,
		Bus:      bus,
		Broker:   broker,
		Sessions: sessions,
		Tokens:   tokens,
		Mailer:   mailer,
	}

	return tenant, nil
}

type tenants map[string]string

func (t *tenants) Set(value string) error {
	if value == "" {
		return nil
	}

	return json.Unmarshal([]byte(value), t)
}

func (t tenants) String() string {
	if t == nil {
		return ""
	}

	b, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	return string(b)
}
