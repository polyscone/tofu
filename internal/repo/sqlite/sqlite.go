package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/repo"
)

//go:embed "migrations"
var migrations embed.FS

const driverName = "sqlite3_custom"

func init() {
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			_, err := conn.Exec(`
				PRAGMA encoding = 'UTF-8';
				PRAGMA busy_timeout = 10000;
				PRAGMA temp_store = MEMORY;
				PRAGMA cache_size = 50000;
				PRAGMA journal_mode = WAL;
				PRAGMA foreign_keys = ON;
				PRAGMA secure_delete = ON;
				PRAGMA synchronous = NORMAL;
			`, nil)

			return errors.Tracef(err)
		},
	})
}

type Kind string

const (
	KindFile   Kind = "file"
	KindMemory Kind = "memory"
)

var databases = struct {
	mu   sync.RWMutex
	data map[string]*sql.DB
}{data: make(map[string]*sql.DB)}

func Open(ctx context.Context, kind Kind, filename string) (*sql.DB, error) {
	var dsn string
	switch kind {
	case KindFile:
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Tracef(err)
		}
		dsn = filename

	case KindMemory:
		// Because Go implements a connection pool it means that when the
		// standard library opens a new connection any in-memory tables etc.
		// will look like they've been lost
		//
		// To get around this it is possible to use a DSN like:
		//
		//     file::memory:?cache=shared
		//
		// References:
		// - https://www.sqlite.org/sharedcache.html#shared_cache_and_in_memory_databases
		// - https://www.sqlite.org/inmemorydb.html#sharedmemdb
		//
		// Using file::memory: with a shared cache would allow each connection
		// in the connection pool to connect to the same in-memory database
		// and share a cache, rather than having their own private one each
		//
		// To allow for many different in-memory connections we can further
		// expand this to take a file name with a mode parameter:
		//
		//     file:<name>?cache=shared&mode=memory
		//
		// All of this together prevents different connections to the same
		// in-memory database from seeing different states, and still allows
		// for unique in-memory databases when required
		dsn = "file:" + filename + "?cache=shared&mode=memory"

	default:
		panic(fmt.Sprintf("unknown sqlite connection kind %q", kind))
	}

	databases.mu.RLock()
	if db, ok := databases.data[dsn]; ok {
		databases.mu.RUnlock()

		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}
	}
	databases.mu.RUnlock()

	databases.mu.Lock()
	defer databases.mu.Unlock()

	db, ok := databases.data[dsn]
	if !ok {
		_db, err := sql.Open(driverName, dsn)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		db = _db
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Tracef(err)
	}

	databases.data[dsn] = db

	return db, nil
}

func OpenInMemoryTestDatabase(ctx context.Context) *sql.DB {
	randomName := errors.Must(uuid.NewV4()).String() + ".db"

	return errors.Must(Open(ctx, KindMemory, randomName))
}

func migrate(ctx context.Context, tx *sql.Tx, name string, migrations []string) error {
	if len(migrations) == 0 {
		return nil
	}

	stmt := `
		CREATE TABLE IF NOT EXISTS _migrations (
			name       TEXT PRIMARY KEY NOT NULL,
			version    INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME
		);

		INSERT OR IGNORE INTO _migrations (name, version) VALUES (:name, 0);
	`
	args := []any{sql.Named("name", name)}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return errors.Tracef(err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM _migrations;").Scan(&count); err != nil {
		return errors.Tracef(err)
	}

	var version int
	stmt = "SELECT version FROM _migrations WHERE name = :name;"
	args = []any{sql.Named("name", name)}
	if err := tx.QueryRowContext(ctx, stmt, args...).Scan(&version); err != nil {
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

	if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys = OFF;"); err != nil {
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
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return errors.Tracef(err)
		}

		migration++
	}

	// If the migration number is greater than the starting version then
	// that means we must have executed some migration strings so we
	// should attempt to set the migration version to the new number
	if migration > version {
		stmt := `
			UPDATE _migrations SET
				version = :version,
				updated_at = :updated_at
			WHERE name = :name;
		`
		args := []any{
			sql.Named("version", migration),
			sql.Named("updated_at", time.Now().UTC()),
			sql.Named("name", name),
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return errors.Tracef(err)
		}
	}

	_, err := tx.ExecContext(ctx, "PRAGMA foreign_keys = ON;")

	return errors.Tracef(err)
}

func migrateFS(ctx context.Context, db *sql.DB, name string, fsys fs.FS) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	files, err := fs.ReadDir(fsys, ".")
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

		b, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return errors.Tracef(err)
		}

		queries[i] = string(b)
	}

	if err := migrate(ctx, tx, name, queries); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func repoerr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return repo.ErrNotFound
	}

	if msg := strings.ToLower(err.Error()); strings.Contains(msg, "unique constraint failed") {
		return repo.ErrConflict
	}

	return err
}

func whereSQL(where []string) string {
	if len(where) == 0 {
		return ""
	}

	return "WHERE " + strings.Join(where, " AND ")
}

func pageLimitOffset(page, size int) (int, int) {
	limit := size
	offset := (page - 1) * size

	return limit, offset
}

func limitOffsetSQL(limit, offset int) string {
	switch {
	case limit > 0 && offset > 0:
		return fmt.Sprintf("LIMIT %v offset %v", limit, offset)

	case limit > 0:
		return fmt.Sprintf("LIMIT %v", limit)

	case offset > 0:
		return fmt.Sprintf("OFFSET %v", offset)
	}

	return ""
}

type NullTime time.Time

func (n NullTime) String() string {
	return time.Time(n).String()
}

func (n *NullTime) UTC() *NullTime {
	if n == nil {
		return nil
	}

	utc := (*time.Time)(n).UTC()

	return (*NullTime)(&utc)
}

func (n *NullTime) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*(*time.Time)(n) = time.Time{}

	case time.Time:
		*(*time.Time)(n) = value

	case *time.Time:
		*(*time.Time)(n) = *value

	case string:
		var err error
		*(*time.Time)(n), err = time.Parse(time.RFC3339, value)
		if err != nil {
			return errors.Tracef(err)
		}

	default:
		return errors.Tracef("NullTime: cannot scan to time.Time: %T", value)
	}

	return nil
}

func (n *NullTime) Value() (driver.Value, error) {
	if n == nil || (*time.Time)(n).IsZero() {
		return nil, nil
	}

	return (*time.Time)(n).Format(time.RFC3339), nil
}

type DB struct {
	db *sql.DB
}

func newDB(db *sql.DB) *DB {
	return &DB{db: db}
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	return &Tx{tx: tx}, nil
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := db.db.ExecContext(ctx, query, args...)
	if err != nil {
		return res, errors.Tracef(repoerr(err))
	}

	return res, nil
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Tracef(repoerr(err))
	}

	return &Rows{rows: rows}, nil
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	row := db.db.QueryRowContext(ctx, query, args...)

	return &Row{row: row}
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

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := tx.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return res, errors.Tracef(repoerr(err))
	}

	return res, nil
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := tx.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Tracef(repoerr(err))
	}

	return &Rows{rows: rows}, nil
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	row := tx.tx.QueryRowContext(ctx, query, args...)

	return &Row{row: row}
}

type Row struct {
	row *sql.Row
}

func (r *Row) Err() error {
	return errors.Tracef(repoerr(r.row.Err()))
}

func (r *Row) Scan(dst ...any) error {
	return errors.Tracef(repoerr(r.row.Scan(dst...)))
}

type Rows struct {
	rows *sql.Rows
}

func (rs *Rows) Close() error {
	return errors.Tracef(repoerr(rs.rows.Close()))
}

func (rs *Rows) Err() error {
	return errors.Tracef(repoerr(rs.rows.Err()))
}

func (rs *Rows) Next() bool {
	return rs.rows.Next()
}

func (rs *Rows) Scan(dst ...any) error {
	return errors.Tracef(repoerr(rs.rows.Scan(dst...)))
}
