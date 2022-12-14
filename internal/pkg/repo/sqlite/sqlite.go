package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

const driver = "sqlite3_custom"

func init() {
	sql.Register(driver, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			pragmas := `
				PRAGMA encoding = 'UTF-8';
				PRAGMA busy_timeout = 10000;
				PRAGMA temp_store = MEMORY;
				PRAGMA cache_size = 50000;
				PRAGMA journal_mode = WAL;
				PRAGMA foreign_keys = ON;
				PRAGMA secure_delete = ON;
				PRAGMA synchronous = NORMAL;
			`

			_, err := conn.Exec(pragmas, nil)

			return errors.Tracef(err)
		},
	})
}

type ReadDirFileFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type Querier interface {
	Exec(ctx context.Context, query string, args ...Args) (sql.Result, error)
	Query(ctx context.Context, query string, args ...Args) (*Rows, error)
	QueryRow(ctx context.Context, query string, args ...Args) *Row
}

type Kind string

const (
	KindFile   Kind = "file"
	KindMemory Kind = "memory"
)

var databases = struct {
	mu   sync.RWMutex
	data map[string]*DB
}{data: make(map[string]*DB)}

func Open(ctx context.Context, kind Kind, filename string) (*DB, error) {
	var dsn string
	switch kind {
	case KindFile:
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Tracef(err)
		}
		dsn = filename

	case KindMemory:
		// Due to the way Go handles connections in the standard library, in-memory SQLite
		// databases have some problems with the wrong connection being used in some cases
		//
		// To get around this problem we need to make sure the cache option for SQLite is
		// set to shared, so that even if a different connection is used in the standard
		// library's connection pool the same cache will still be used for that in-memory database
		//
		// This will prevent errors like the database thinking a table doesn't exist when it does etc.
		//
		// The cache is only shared between in-memory databases that have the same name, so we
		// can still use different in-memory databases by setting the file name to something
		// unique when required
		dsn = "file:" + filename + ":memory?mode=memory&cache=shared"

	default:
		panic(fmt.Sprintf("unknown sqlite connection kind %q", kind))
	}

	databases.mu.RLock()
	if db, ok := databases.data[dsn]; ok {
		databases.mu.RUnlock()

		if err := db.Ping(ctx); err == nil {
			return db, nil
		}
	}
	databases.mu.RUnlock()

	_db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	db := &DB{db: _db}

	if err := db.Ping(ctx); err != nil {
		return nil, errors.Tracef(err)
	}

	databases.mu.Lock()
	defer databases.mu.Unlock()

	databases.data[dsn] = db

	return db, nil
}

func OpenInMemoryTestDatabase(ctx context.Context) *DB {
	randomName := errors.Must(uuid.NewV4()).String() + ".db"

	return errors.Must(Open(ctx, KindMemory, randomName))
}

func migrate(ctx context.Context, tx *Tx, name string, migrations []string) error {
	if len(migrations) == 0 {
		return nil
	}

	stmt, args := `
		CREATE TABLE IF NOT EXISTS _migrations (
			name       TEXT PRIMARY KEY NOT NULL,
			version    INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME
		);

		INSERT OR IGNORE INTO _migrations (name, version) VALUES (:name, 0);
	`, Args{"name": name}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return errors.Tracef(err)
	}

	var count int
	if err := tx.QueryRow(ctx, "SELECT COUNT(1) FROM _migrations;").Scan(&count); err != nil {
		return errors.Tracef(err)
	}

	var version int
	stmt, args = "SELECT version FROM _migrations WHERE name = :name;", Args{"name": name}
	if err := tx.QueryRow(ctx, stmt, args).Scan(&version); err != nil {
		return errors.Tracef(err)
	}

	if version < 0 {
		return errors.Tracef("want current migration to be version 0 or more; got %v", version)
	}
	migration := version
	nm := len(migrations)

	// If the number of migration strings is less than the version then we must have
	// lost some migrations and the data cannot be trusted
	if nm < version {
		return errors.Tracef("want at least %v migration strings; got %v", version, nm)
	}

	// If the version is the same as the number of migration strings then we must be up to date
	if nm == version {
		return nil
	}

	if _, err := tx.Exec(ctx, "PRAGMA foreign_keys = OFF;"); err != nil {
		return errors.Tracef(err)
	}

	for i, stmt := range migrations {
		if i < version {
			continue
		}

		// If the migration file is empty then don't waste the
		// time trying to execute a query
		if stmt = strings.TrimSpace(stmt); stmt == "" {
			continue
		}
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return errors.Tracef(err)
		}

		migration++
	}

	// If the migration number is greater than the starting version then
	// that means we must have executed some migration strings so we
	// should attempt to set the migration version to the new number
	if migration > version {
		stmt, args := `
			UPDATE _migrations SET
				version = :version,
				updated_at = :updated_at
			WHERE name = :name;
		`, Args{
			"version":    migration,
			"updated_at": time.Now().UTC(),
			"name":       name,
		}
		if _, err := tx.Exec(ctx, stmt, args); err != nil {
			return errors.Tracef(err)
		}
	}

	_, err := tx.Exec(ctx, "PRAGMA foreign_keys = ON;")

	return errors.Tracef(err)
}

type Args map[string]any

func argsSlice(argMaps []Args) []any {
	switch len(argMaps) {
	case 0:
		return nil

	case 1:
		var args []any
		for param, arg := range argMaps[0] {
			args = append(args, sql.Named(param, arg))
		}

		return args

	default:
		panic("expected no more than one args map")
	}
}

type DB struct {
	db *sql.DB
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

func (db *DB) Begin(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	return &Tx{tx: tx}, nil
}

func (db *DB) Exec(ctx context.Context, query string, args ...Args) (sql.Result, error) {
	res, err := db.db.ExecContext(ctx, query, argsSlice(args)...)
	if err != nil {
		return res, errors.Tracef(asRepoError(err))
	}

	return res, nil
}

func (db *DB) Query(ctx context.Context, query string, args ...Args) (*Rows, error) {
	rows, err := db.db.QueryContext(ctx, query, argsSlice(args)...)
	if err != nil {
		return nil, errors.Tracef(asRepoError(err))
	}

	return &Rows{rows: rows}, nil
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...Args) *Row {
	row := db.db.QueryRowContext(ctx, query, argsSlice(args)...)

	return &Row{row: row}
}

func (db *DB) MigrateFS(ctx context.Context, prefix string, fsys ReadDirFileFS) error {
	tx, err := db.Begin(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	groups, err := fsys.ReadDir(".")
	if err != nil {
		return errors.Tracef(err)
	}

	for _, group := range groups {
		if !group.IsDir() {
			continue
		}

		dir := group.Name()
		name := fmt.Sprintf("%v_%v", prefix, dir)

		files, err := fsys.ReadDir(dir)
		if err != nil {
			return errors.Tracef(err)
		}

		queries := make([]string, len(files))
		for i, f := range files {
			filename := f.Name()

			if f.IsDir() {
				return errors.Tracef("want file; got directory %q", filename)
			}

			if filename[:4] != fmt.Sprintf("%04d", i+1) {
				return errors.Tracef("want file beginning with %04d; got %q", i+1, filename)
			}

			fp := path.Join(dir, filename)
			b, err := fsys.ReadFile(fp)
			if err != nil {
				return errors.Tracef(err)
			}

			queries[i] = string(b)
		}

		if err := migrate(ctx, tx, name, queries); err != nil {
			return errors.Tracef(err)
		}
	}

	return errors.Tracef(tx.Commit())
}

type Tx struct {
	tx *sql.Tx
}

func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

func (tx *Tx) Exec(ctx context.Context, query string, args ...Args) (sql.Result, error) {
	res, err := tx.tx.ExecContext(ctx, query, argsSlice(args)...)
	if err != nil {
		return res, errors.Tracef(asRepoError(err))
	}

	return res, nil
}

func (tx *Tx) Query(ctx context.Context, query string, args ...Args) (*Rows, error) {
	rows, err := tx.tx.QueryContext(ctx, query, argsSlice(args)...)
	if err != nil {
		return nil, errors.Tracef(asRepoError(err))
	}

	return &Rows{rows: rows}, nil
}

func (tx *Tx) QueryRow(ctx context.Context, query string, args ...Args) *Row {
	row := tx.tx.QueryRowContext(ctx, query, argsSlice(args)...)

	return &Row{row: row}
}

type Row struct {
	row *sql.Row
}

func (r *Row) Err() error {
	return errors.Tracef(asRepoError(r.row.Err()))
}

func (r *Row) Scan(dst ...any) error {
	return errors.Tracef(asRepoError(r.row.Scan(dst...)))
}

type Rows struct {
	rows *sql.Rows
}

func (rs *Rows) Close() error {
	return errors.Tracef(asRepoError(rs.rows.Close()))
}

func (rs *Rows) Err() error {
	return errors.Tracef(asRepoError(rs.rows.Err()))
}

func (rs *Rows) Next() bool {
	return rs.rows.Next()
}

func (rs *Rows) Scan(dst ...any) error {
	return errors.Tracef(asRepoError(rs.rows.Scan(dst...)))
}

func asRepoError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())

	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = repo.ErrNotFound

	case strings.Contains(msg, "unique constraint failed"):
		err = repo.ErrConflict
	}

	return err
}
