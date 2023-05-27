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

	// We check for an existing connection pool here because another goroutine
	// could have already connected whilst other locks were waiting
	if db, ok := databases.data[dsn]; ok {
		// It's possible that we find an existing connection pool that failed
		// to ping, in which case we only want to return it if we know the
		// connection is still alive, so we try to ping again to be sure
		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}

		// If we reach this point then it's because we've failed to ping the
		// database a couple of times and we need to replace the connection pool
		// so we attempt to close before replacing
		db.Close()
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, errors.Tracef(err)
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
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIMESTAMP)),
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
			sql.Named("updated_at", Time(time.Now().UTC())),
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

type Time time.Time

func (t Time) String() string {
	return time.Time(t).String()
}

func (t Time) UTC() Time {
	return Time(time.Time(t).UTC())
}

func (t *Time) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*t = Time(time.Time{})

	case time.Time:
		*t = Time(value)

	case *time.Time:
		*t = Time(*value)

	case string:
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return errors.Tracef(err)
		}

		*t = Time(parsed)

	default:
		return errors.Tracef("Time: cannot scan to time.Time: %T", value)
	}

	return nil
}

func (t Time) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return time.Time{}.Format(time.RFC3339), nil
	}

	return time.Time(t).Format(time.RFC3339), nil
}

type NullTime time.Time

func (t NullTime) String() string {
	return time.Time(t).String()
}

func (t NullTime) UTC() NullTime {
	return NullTime(time.Time(t).UTC())
}

func (t *NullTime) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*t = NullTime(time.Time{})

	case time.Time:
		*t = NullTime(value)

	case *time.Time:
		*t = NullTime(*value)

	case string:
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return errors.Tracef(err)
		}

		*t = NullTime(parsed)

	default:
		return errors.Tracef("NullTime: cannot scan to time.Time: %T", value)
	}

	return nil
}

func (t NullTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}

	return time.Time(t).Format(time.RFC3339), nil
}

func validateArg(arg any) error {
	switch arg := arg.(type) {
	case time.Time, *time.Time, **time.Time:
		return errors.Tracef(
			"cannot use %T as an arg; convert to one of: %T, %T, %T, or %T instead",
			arg, Time{}, &Time{}, NullTime{}, &NullTime{},
		)

	case sql.NamedArg:
		return validateArg(arg.Value)

	default:
		return nil
	}
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
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, errors.Tracef(err)
		}
	}

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
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return &Row{err: errors.Tracef(err)}
		}
	}

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
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, errors.Tracef(err)
		}
	}

	res, err := tx.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return res, errors.Tracef(repoerr(err))
	}

	return res, nil
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, errors.Tracef(err)
		}
	}

	rows, err := tx.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Tracef(repoerr(err))
	}

	return &Rows{rows: rows}, nil
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return &Row{err: errors.Tracef(err)}
		}
	}

	row := tx.tx.QueryRowContext(ctx, query, args...)

	return &Row{row: row}
}

type Row struct {
	err error
	row *sql.Row
}

func (r *Row) Err() error {
	if r.err != nil {
		return errors.Tracef(r.err)
	}

	return errors.Tracef(repoerr(r.row.Err()))
}

func (r *Row) Scan(dst ...any) error {
	if r.err != nil {
		return errors.Tracef(r.err)
	}

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
