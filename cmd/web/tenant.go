package main

import (
	"context"
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

type repo struct {
	account *sqlite.AccountRepo
	system  *sqlite.SystemRepo
	web     *sqlite.WebRepo
}

var cache = struct {
	mu     sync.Mutex
	repos  map[string]repo
	sqlite map[string]*sqlite.DB
	mailer smtp.Mailer
}{
	repos:  make(map[string]repo),
	sqlite: make(map[string]*sqlite.DB),
}

// newTenant returns a tenant where the hostname is mapped to a shared alias.
//
// Tenants share repositories along with their underlying database connection
// pools based on the alias, and all tenants share an SMTP mailer regardless
// of alias.
//
// Every tenant gets its own event broker regardless of alias so that different
// adapters, even for those handling the same hostname, can respond to
// application events differently if required.
func newTenant(hostname string) (*handler.Tenant, error) {
	ctx := context.Background()

	data, ok := tenants[hostname]
	if !ok {
		return nil, fmt.Errorf("find tenant %v: %w", hostname, web.ErrTenantNotFound)
	}
	if data.Alias == "" {
		return nil, fmt.Errorf("alias name for the tenant %v is empty", hostname)
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	repo, ok := cache.repos[data.Alias]
	if !ok {
		var err error

		sqliteDB := cache.sqlite[data.Alias]
		if sqliteDB == nil {
			p := filepath.Join(opts.data, data.Alias, "main.sqlite")
			sqliteDB, err = sqlite.Open(ctx, sqlite.KindFile, p)
			if err != nil {
				return nil, fmt.Errorf("open database: %w", err)
			}

			cache.sqlite[data.Alias] = sqliteDB
		}

		repo.account, err = sqlite.NewAccountRepo(ctx, sqliteDB, app.SignInThrottleTTL)
		if err != nil {
			return nil, fmt.Errorf("new account repo: %w", err)
		}

		repo.system, err = sqlite.NewSystemRepo(ctx, sqliteDB)
		if err != nil {
			return nil, fmt.Errorf("new system repo: %w", err)
		}

		repo.web, err = sqlite.NewWebRepo(ctx, sqliteDB, app.SessionTTL)
		if err != nil {
			return nil, fmt.Errorf("new web repo: %w", err)
		}

		cache.repos[data.Alias] = repo
	}

	if cache.mailer == nil {
		var err error
		cache.mailer, err = smtp.NewMailClient("localhost", 25)
		if err != nil {
			return nil, fmt.Errorf("new SMTP client: %w", err)
		}
	}

	var svc handler.Svc
	var err error

	broker := event.NewMemoryBroker()

	svc.Account, err = account.NewService(broker, repo.account, hasher)
	if err != nil {
		return nil, fmt.Errorf("new account service: %w", err)
	}

	svc.System, err = system.NewService(broker, repo.system)
	if err != nil {
		return nil, fmt.Errorf("new system service: %w", err)
	}

	tenant := handler.Tenant{
		Kind:     data.Kind,
		Dev:      opts.dev,
		Insecure: opts.server.insecure,
		Proxies:  opts.server.proxies,
		Broker:   broker,
		Email:    cache.mailer,
		Svc:      svc,
		Repo: handler.Repo{
			Account: repo.account,
			System:  repo.system,
			Web:     repo.web,
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

		if alias == "" && len(tenant.Hostnames) > 0 {
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
