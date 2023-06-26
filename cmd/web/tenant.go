package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/smtp"
	"github.com/polyscone/tofu/internal/repository/sqlite"
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
		return nil, fmt.Errorf("find tenant %v: %w", hostname, web.ErrTenantNotFound)
	}
	if data.Alias == "" {
		return nil, fmt.Errorf("alias name for the tenant %v is empty", hostname)
	}

	databases.mu.Lock()
	defer databases.mu.Unlock()

	db := databases.data[data.Alias]
	if db == nil {
		var err error
		p := filepath.Join(opts.data, data.Alias, "main.sqlite")
		db, err = sqlite.Open(ctx, sqlite.KindFile, p)
		if err != nil {
			return nil, fmt.Errorf("open database: %w", err)
		}

		databases.data[data.Alias] = db
	}

	broker := event.NewMemoryBroker()

	accountRepo, err := sqlite.NewAccountRepo(ctx, db, app.SignInThrottleTTL)
	if err != nil {
		return nil, fmt.Errorf("new account repo: %w", err)
	}

	systemRepo, err := sqlite.NewSystemRepo(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("new system repo: %w", err)
	}

	webRepo, err := sqlite.NewWebRepo(ctx, db, app.SessionTTL)
	if err != nil {
		return nil, fmt.Errorf("new web repo: %w", err)
	}

	mailer, err := smtp.NewMailClient("localhost", 25)
	if err != nil {
		return nil, fmt.Errorf("new SMTP client: %w", err)
	}

	accountService, err := account.NewService(broker, accountRepo, hasher)
	if err != nil {
		return nil, fmt.Errorf("new account service: %w", err)
	}

	systemService, err := system.NewService(broker, systemRepo)
	if err != nil {
		return nil, fmt.Errorf("new system service: %w", err)
	}

	tenant := handler.Tenant{
		Kind:     data.Kind,
		Dev:      opts.dev,
		Insecure: opts.server.insecure,
		Proxies:  opts.server.proxies,
		Broker:   broker,
		Email: handler.Email{
			Mailer: mailer,
		},
		Account: accountService,
		System:  systemService,
		Repo: handler.Repo{
			Account: accountRepo,
			System:  systemRepo,
			Web:     webRepo,
		},
	}

	return &tenant, nil
}

type Tenant struct {
	Alias      string            `json:"-"`
	Kind       string            `json:"-"`
	Hostnames  map[string]string `json:"hostnames"`
	IsDisabled bool              `json:"isDisabled"`
}

func initTenants(tenantsPath string) error {
	tenants = make(map[string]Tenant)

	f, err := os.OpenFile(tenantsPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("open or create tenants file: %w", err)
	}
	err = func() error {
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("tenants file stat info: %w", err)
		}

		if info.Size() == 0 {
			example := map[string]Tenant{
				"example": {
					Hostnames: map[string]string{
						"www.example.com": "site",
						"app.example.com": "pwa",
					},
					IsDisabled: true,
				},
			}

			b, err := json.MarshalIndent(example, "", "\t")
			if err != nil {
				return fmt.Errorf("marshal example tenant file data: %w", err)
			}

			if _, err := f.Write(b); err != nil {
				return fmt.Errorf("write example tenant file data: %w", err)
			}
		}

		return nil
	}()
	if err != nil {
		return err
	}

	tenantsData, err := os.ReadFile(tenantsPath)
	if err != nil {
		return fmt.Errorf("read tenants file: %w", err)
	}

	data := make(map[string]Tenant)
	if err := json.Unmarshal([]byte(tenantsData), &data); err != nil {
		return fmt.Errorf("unmarshal tenant data: %w", err)
	}

	var errs errsx.Map
	for alias, tenant := range data {
		if len(tenant.Hostnames) == 0 {
			errs.Set(alias+".hostnames", "must be populated with at least one hostname")
		}

		if alias == "" && len(tenant.Hostnames) != 0 {
			for hostname := range tenant.Hostnames {
				errs.Set("hostname "+hostname, "alias cannot be empty")
			}

			continue
		}

		for hostname, kind := range tenant.Hostnames {
			if dupe, ok := tenants[hostname]; ok {
				errs.Set(hostname, fmt.Sprintf("cannot associate with %q; already associated with %q", alias, dupe.Alias))
			}

			tenant.Alias = alias
			tenant.Kind = kind

			if !tenant.IsDisabled {
				tenants[hostname] = tenant
			}
		}
	}
	if errs != nil {
		return fmt.Errorf("tenant configuration errors: %w", errs)
	}

	return nil
}
