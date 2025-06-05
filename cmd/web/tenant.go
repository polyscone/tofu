package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/event"
	"github.com/polyscone/tofu/internal/smtp"
	"github.com/polyscone/tofu/repo"
	"github.com/polyscone/tofu/repo/sqlite"
	"github.com/polyscone/tofu/web"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
)

var tenants = make(map[string]Tenant)

type repos struct {
	account *repo.Account
	system  *repo.System
	web     *repo.Web
}

var cache = struct {
	mu         sync.Mutex
	recoverers map[string]*RecoveryService
	repos      map[string]repos
	sqlite     map[string]*sqlite.DB
	mailers    map[string]smtp.Mailer
	loggers    map[string]*slog.Logger
	metrics    map[string]*expvar.Map
}{
	recoverers: make(map[string]*RecoveryService),
	repos:      make(map[string]repos),
	sqlite:     make(map[string]*sqlite.DB),
	mailers:    make(map[string]smtp.Mailer),
	loggers:    make(map[string]*slog.Logger),
	metrics:    make(map[string]*expvar.Map),
}

func closeCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	for alias, db := range cache.sqlite {
		if db == nil {
			continue
		}

		if err := db.Close(); err != nil {
			slog.Error("close SQLite database connection", "alias", alias, "error", err)
		}
	}
}

func sqliteOpen(ctx context.Context, data *Tenant, metrics *expvar.Map, recovery *RecoveryService, p string) (*sqlite.DB, string, error) {
	key := data.Name + "." + p
	name := filepath.Base(p)
	sqliteDB := cache.sqlite[key]
	if sqliteDB == nil {

		var dbMetrics *expvar.Map
		if metrics != nil {
			metricsName := strings.TrimSuffix(name, ".sqlite")
			metricsKey := fmt.Sprintf("database.SQLite %v", metricsName)

			var ok bool
			dbMetrics, ok = metrics.Get(metricsKey).(*expvar.Map)
			if !ok {
				dbMetrics = &expvar.Map{}

				dbMetrics.Set("stats", expvar.Func(func() any {
					if sqliteDB.DB == nil {
						return sql.DBStats{}
					}

					return sqliteDB.Stats()
				}))

				metrics.Set(metricsKey, dbMetrics)
			}
		}

		var err error
		sqliteDB, err = sqlite.Open(ctx, sqlite.KindFile, p, dbMetrics, nil)
		if err != nil {
			return nil, "", fmt.Errorf("open SQLite database: %w", err)
		}

		if recovery != nil {
			recovery.sqliteDBs[name] = sqliteDB
		}

		cache.sqlite[key] = sqliteDB
	}

	return sqliteDB, name, nil
}

// newTenant returns a tenant where the host is mapped to a shared name.
//
// Tenants share repositories along with their underlying database connection
// pools based on the name, and all tenants share an SMTP mailer regardless
// of name.
//
// Every tenant gets its own event broker regardless of name so that different
// adapters, even for those handling the same host, can respond to
// application events differently if required.
func newTenant(host string) (*handler.Tenant, error) {
	ctx := context.Background()

	data, ok := tenants[host]
	if !ok {
		return nil, fmt.Errorf("find tenant %v: %w", host, web.ErrTenantNotFound)
	}
	if data.Name == "" {
		return nil, fmt.Errorf("name for the tenant %v is empty", host)
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	metrics, ok := cache.metrics[data.Name]
	if !ok {
		metrics = expvar.NewMap("tenant." + data.Name)

		cache.metrics[data.Name] = metrics
	}

	logger, ok := cache.loggers[data.Name]
	if !ok {
		var err error
		logger, err = opts.log.style.NewLogger(nil)
		if err != nil {
			return nil, fmt.Errorf("new logger: %w", err)
		}

		logger = logger.With("app", data.Name)
		logger = logger.With("kind", data.Kind)

		cache.loggers[data.Name] = logger
	}

	dataDir := filepath.Join(opts.dataDir, data.Name)

	recovery, ok := cache.recoverers[data.Name]
	if !ok {
		recovery = NewRecoveryService(dataDir, logger)

		cache.recoverers[data.Name] = recovery
	}

	repos, ok := cache.repos[data.Name]
	if !ok {
		// Account repo
		{
			p := filepath.Join(dataDir, "account.sqlite")
			sqliteDB, _, err := sqliteOpen(ctx, &data, metrics, recovery, p)
			if err != nil {
				return nil, fmt.Errorf("open SQLite database: %w", err)
			}

			repos.account, err = repo.NewAccount(ctx, sqliteDB, app.SignInThrottleTTL)
			if err != nil {
				return nil, fmt.Errorf("new account repo: %w", err)
			}
		}

		// System repo
		{
			p := filepath.Join(dataDir, "system.sqlite")
			sqliteDB, name, err := sqliteOpen(ctx, &data, metrics, recovery, p)
			if err != nil {
				return nil, fmt.Errorf("open SQLite database: %w", err)
			}

			repos.system, err = repo.NewSystem(ctx, sqliteDB)
			if err != nil {
				return nil, fmt.Errorf("new system repo: %w", err)
			}

			recovery.system = repos.system

			recovery.sqliteDBKinds[name] = "system"
		}

		// Web repo
		{
			p := filepath.Join(dataDir, "web.sqlite")
			sqliteDB, name, err := sqliteOpen(ctx, &data, metrics, recovery, p)
			if err != nil {
				return nil, fmt.Errorf("open SQLite database: %w", err)
			}

			repos.web, err = repo.NewWeb(ctx, sqliteDB, app.SessionTTL)
			if err != nil {
				return nil, fmt.Errorf("new web repo: %w", err)
			}

			recovery.sqliteDBKinds[name] = "web"
		}

		cache.repos[data.Name] = repos
	}

	mailer, ok := cache.mailers[data.Name]
	if !ok {
		var err error
		mailer, err = smtp.NewClient(logger, &smtpConfig{
			envelopeEmail: data.SMTPEnvelopeEmail,
			system:        repos.system,
		})
		if err != nil {
			return nil, fmt.Errorf("new dynamic SMTP client: %w", err)
		}

		cache.mailers[data.Name] = mailer
	}

	var svc handler.Svc
	var err error

	broker := event.NewMemoryBroker()

	svc.Account, err = account.NewService(broker, repos.account, hasher, data.Kind)
	if err != nil {
		return nil, fmt.Errorf("new account service: %w", err)
	}

	svc.System, err = system.NewService(broker, repos.system)
	if err != nil {
		return nil, fmt.Errorf("new system service: %w", err)
	}

	var permissions []account.Permission
	for _, group := range guard.PermissionGroups {
		for _, p := range group.Permissions {
			permission, err := account.NewPermission(p.Name)
			if err != nil {
				return nil, fmt.Errorf("new permission: %w", err)
			}

			permissions = append(permissions, permission)
		}
	}

	superRole := account.NewRole("Super", "Has full access to the system; can't be edited or deleted.", permissions)

	role, err := repos.account.FindRoleByName(ctx, superRole.Name)
	switch {
	case err == nil:
		superRole.ID = role.ID

		if err := repos.account.SaveRole(ctx, superRole); err != nil {
			return nil, fmt.Errorf("save super role: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		if err := repos.account.AddRole(ctx, superRole); err != nil {
			return nil, fmt.Errorf("add super role: %w", err)
		}

	default:
		return nil, fmt.Errorf("find role by name: %w", err)
	}

	tenant := handler.Tenant{
		Key:               host + "." + data.Name,
		Kind:              data.Kind,
		Hosts:             data.Hosts,
		DataDir:           dataDir,
		Dev:               opts.dev,
		IPWhitelist:       opts.server.ipWhitelist,
		Proxies:           opts.server.proxies,
		SMTPEnvelopeEmail: data.SMTPEnvelopeEmail,
		Broker:            broker,
		Email:             mailer,
		Logger:            logger,
		Metrics:           metrics,
		Svc:               svc,
		Repo: handler.Repo{
			Account: repos.account,
			System:  repos.system,
			Web:     repos.web,
		},
		Recovery:  recovery,
		SuperRole: superRole,
	}

	return &tenant, nil
}

type Tenant struct {
	Name              string              `json:"-"`
	Kind              string              `json:"-"`
	Hosts             map[string]string   `json:"hosts"`
	Aliases           map[string][]string `json:"aliases"`
	SMTPEnvelopeEmail string              `json:"smtpEnvelopeEmail"`
	IsDisabled        bool                `json:"isDisabled"`
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
					Hosts: map[string]string{
						"site": "www.example.com",
						"pwa":  "app.example.com",
					},
					Aliases: map[string][]string{
						"site": {"localhost", "localhost:8080"},
					},
					SMTPEnvelopeEmail: "",
					IsDisabled:        true,
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
	for name, tenant := range data {
		if len(tenant.Hosts) == 0 {
			errs.Set(name+".hosts", "must be populated with at least one host")
		}

		if name == "" && len(tenant.Hosts) > 0 {
			for host := range tenant.Hosts {
				errs.Set("host "+host, "name cannot be empty")
			}

			continue
		}

		for kind, host := range tenant.Hosts {
			if dupe, ok := tenants[host]; ok {
				errs.Set(host, fmt.Sprintf("cannot associate with %q; already associated with %q", name, dupe.Name))
			}

			tenant.Name = name
			tenant.Kind = kind

			if !tenant.IsDisabled {
				tenants[host] = tenant
			}
		}

		for kind, hosts := range tenant.Aliases {
			for _, host := range hosts {
				if dupe, ok := tenants[host]; ok {
					errs.Set(host, fmt.Sprintf("cannot associate with %q; already associated with %q", name, dupe.Name))
				}

				tenant.Name = name
				tenant.Kind = kind

				if !tenant.IsDisabled {
					tenants[host] = tenant
				}
			}
		}
	}
	if errs != nil {
		return fmt.Errorf("tenant configuration errors: %w", errs)
	}

	return nil
}
