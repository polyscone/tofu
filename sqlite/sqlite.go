package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"expvar"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/uuid"
)

//go:embed "all:migrations"
var migrations embed.FS

const driverName = "sqlite3_custom"

func init() {
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			_, err := conn.Exec(`
				PRAGMA encoding = 'UTF-8';
				PRAGMA busy_timeout = 30000;
				PRAGMA temp_store = MEMORY;
				PRAGMA cache_size = 5000;
				PRAGMA journal_mode = WAL;
				PRAGMA foreign_keys = ON;
				PRAGMA secure_delete = ON;
				PRAGMA synchronous = NORMAL;
			`, nil)

			return err
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
	data map[string]*DB
}{data: make(map[string]*DB)}

func Open(ctx context.Context, kind Kind, filename string, metrics *expvar.Map) (*DB, error) {
	var dsn string
	switch kind {
	case KindFile:
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("make directory: %w", err)
		}
		dsn = filename

	case KindMemory:
		// Because Go implements a connection pool it means that when the
		// standard library opens a new connection any in-memory tables etc.
		// will look like they've been lost because they were created on a
		// different connection from the same pool
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
		// expand this to take a file name with a mode parameter instead:
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

		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

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
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

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

	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, repoerr(err)
	}

	db := &DB{
		DB:      sqlDB,
		metrics: metrics,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", repoerr(err))
	}

	databases.data[dsn] = db

	return db, nil
}

func OpenInMemoryTestDatabase(ctx context.Context) *DB {
	randomName := errsx.Must(uuid.NewV4()).String() + ".sqlite"

	return errsx.Must(Open(ctx, KindMemory, randomName, nil))
}

func migrate(ctx context.Context, tx *Tx, name string, migrations []string) error {
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
		return err
	}

	var version int
	stmt = "SELECT version FROM _migrations WHERE name = :name;"
	args = []any{sql.Named("name", name)}
	if err := tx.QueryRowContext(ctx, stmt, args...).Scan(&version); err != nil {
		return err
	}

	if version < 0 {
		return fmt.Errorf("want current migration to be version 0 or more; got %v", version)
	}
	migration := version
	nm := len(migrations)

	// If the number of migration strings is less than the version then we must have
	// lost some migrations and the data cannot be trusted
	if nm < version {
		return fmt.Errorf("want at least %v migration strings; got %v", version, nm)
	}

	// If the version is the same as the number of migration strings then we must be up to date
	if nm == version {
		return nil
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
			return err
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
			return err
		}
	}

	return nil
}

func migrateFS(ctx context.Context, db *DB, name string, fsys fs.FS) error {
	// We get a connection from the pool directly here so we can make
	// sure that pragmas run on this connection will be in effect for the
	// transaction we use for migrations
	//
	// If we use sql.DB.BeginTx directly then the connection used for the
	// transaction could end up being a different one from the pool
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer conn.Close()

	// Pragmas need to be set before the transaction starts, which is why
	// we do it on the connection here
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF;"); err != nil {
		return err
	}

	tx, err := conn.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	files, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}

	queries := make([]string, len(files))
	for i, f := range files {
		filename := f.Name()

		if f.IsDir() {
			return fmt.Errorf("want file; got directory %q", filename)
		}

		if filename[:4] != fmt.Sprintf("%04d", i+1) {
			return fmt.Errorf("want file beginning with %04d; got %q", i+1, filename)
		}

		b, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return fmt.Errorf("read migration file: %w", err)
		}

		queries[i] = string(b)
	}

	if err := migrate(ctx, tx, name, queries); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	// Since pragmas are set outside of the transaction we need to
	// make sure we revert the changes here
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		return err
	}

	return nil
}

type recordKind byte

const (
	recordKindUnknown recordKind = iota
	recordKindRead
	recordKindWrite
)

func recordQuery(metrics *expvar.Map, kind recordKind) func() {
	if metrics == nil {
		return func() {}
	}

	now := time.Now()

	return func() {
		since := time.Since(now).Nanoseconds()

		switch kind {
		case recordKindRead:
			metrics.Add("totalReads", 1)
			metrics.Add("totalReadTime", since)

		case recordKindWrite:
			metrics.Add("totalWrites", 1)
			metrics.Add("totalWriteTime", since)
		}

		metrics.Add("totalQueries", 1)
		metrics.Add("totalQueryTime", since)
	}
}

func repoerr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return app.ErrNotFound
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unique constraint failed"):
		return app.ErrConflict

	case strings.Contains(msg, "login error"):
		return fmt.Errorf("%w: %w", app.ErrRepoLogin, err)
	}

	return err
}

func whereSQL(where []string) string {
	if len(where) == 0 {
		return ""
	}

	return "WHERE (" + strings.Join(where, ") AND (") + ")"
}

func orderBySQL(sorts []string) string {
	if len(sorts) == 0 {
		return ""
	}

	return "ORDER BY " + strings.Join(sorts, ", ")
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

func inSQL[T any](args []T) (string, []any) {
	placeholders := strings.Join(strings.Split(strings.Repeat("?", len(args)), ""), ", ")

	values := make([]any, len(args))
	for i := 0; i < len(values); i++ {
		values[i] = args[i]
	}

	return placeholders, values
}

const RFC3339NanoZero = "2006-01-02T15:04:05.000000000Z07:00"

type Time time.Time

func (t Time) String() string {
	return time.Time(t).String()
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
		parsed, err := time.Parse(RFC3339NanoZero, value)
		if err != nil {
			return fmt.Errorf("parse RFC3339 nano with trailing zeros: %w", err)
		}

		*t = Time(parsed)

	default:
		return fmt.Errorf("%T: cannot scan to time.Time: %T", Time{}, value)
	}

	return nil
}

func (t Time) Value() (driver.Value, error) {
	return time.Time(t).Format(RFC3339NanoZero), nil
}

type NullTime time.Time

func (t NullTime) String() string {
	return time.Time(t).String()
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
		parsed, err := time.Parse(RFC3339NanoZero, value)
		if err != nil {
			return fmt.Errorf("parse RFC3339 nano with trailing zeros: %w", err)
		}

		*t = NullTime(parsed)

	default:
		return fmt.Errorf("%T: cannot scan to time.Time: %T", NullTime{}, value)
	}

	return nil
}

func (t NullTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}

	return time.Time(t).Format(RFC3339NanoZero), nil
}

type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d *Duration) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*d = Duration(0)

	case time.Duration:
		*d = Duration(value)

	case *time.Duration:
		*d = Duration(*value)

	case int64:
		*d = Duration(value)

	case string:
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}

		*d = Duration(parsed)

	default:
		return fmt.Errorf("%T: cannot scan to time.Duration: %T", Duration(0), value)
	}

	return nil
}

func (d Duration) Value() (driver.Value, error) {
	return time.Duration(d).String(), nil
}

type NullDuration time.Duration

func (d NullDuration) String() string {
	return time.Duration(d).String()
}

func (d *NullDuration) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*d = NullDuration(0)

	case time.Duration:
		*d = NullDuration(value)

	case *time.Duration:
		*d = NullDuration(*value)

	case int64:
		*d = NullDuration(value)

	case string:
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}

		*d = NullDuration(parsed)

	default:
		return fmt.Errorf("%T: cannot scan to time.Duration: %T", NullDuration(0), value)
	}

	return nil
}

func (d NullDuration) Value() (driver.Value, error) {
	if d == 0 {
		return nil, nil
	}

	return time.Duration(d).String(), nil
}

func validateArg(arg any) error {
	switch arg := arg.(type) {
	case time.Time, *time.Time, **time.Time,
		sql.NullTime, *sql.NullTime, **sql.NullTime,
		sql.Null[time.Time], *sql.Null[time.Time], **sql.Null[time.Time]:

		return fmt.Errorf(
			"cannot use %T as an arg; convert to one of: %T, %T, %T, or %T instead",
			arg, Time{}, &Time{}, NullTime{}, &NullTime{},
		)

	case time.Duration, *time.Duration, **time.Duration:
		d1 := Duration(0)
		d2 := NullDuration(0)

		return fmt.Errorf(
			"cannot use %T as an arg; convert to one of: %T, %T, %T, or %T instead",
			arg, d1, &d1, d2, &d2,
		)

	case sql.NamedArg:
		return validateArg(arg.Value)

	default:
		return nil
	}
}

type Conn struct {
	*sql.Conn
	metrics *expvar.Map
}

func (c *Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	if c.metrics != nil {
		c.metrics.Add("totalTransactionsBegun", 1)
	}

	_tx := &Tx{Tx: tx, ctx: ctx, now: time.Now(), metrics: c.metrics}

	go _tx.awaitDone()

	return _tx, nil
}

// BeginImmediateTx starts an immediate transaction with "BEGIN IMMEDIATE".
//
// This is a workaround for Go's database/sql package not providing a way to set
// the transaction type per connection.
//
// References:
// - https://github.com/golang/go/issues/19981
// - https://github.com/mattn/go-sqlite3/issues/400
func (c *Conn) BeginImmediateTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := c.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// The returned transaction is a connection from a connection pool, so we
	// can rollback the (by default) "DEFERRED" transaction and start a new
	// "IMMEDIATE" one
	if _, err := tx.ExecContext(ctx, "ROLLBACK; BEGIN IMMEDIATE"); err != nil {
		return nil, err
	}

	if c.metrics != nil {
		c.metrics.Add("totalTransactionsBegun", 1)
	}

	return tx, nil
}

// BeginExclusiveTx starts an exclusive transaction with "BEGIN EXCLUSIVE".
//
// This is a workaround for Go's database/sql package not providing a way to set
// the transaction type per connection.
//
// References:
// - https://github.com/golang/go/issues/19981
// - https://github.com/mattn/go-sqlite3/issues/400
func (c *Conn) BeginExclusiveTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := c.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// The returned transaction is a connection from a connection pool, so we
	// can rollback the (by default) "DEFERRED" transaction and start a new
	// "EXCLUSIVE" one
	if _, err := tx.ExecContext(ctx, "ROLLBACK; BEGIN EXCLUSIVE"); err != nil {
		return nil, err
	}

	if c.metrics != nil {
		c.metrics.Add("totalTransactionsBegun", 1)
	}

	return tx, nil
}

func (c *Conn) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := c.Conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return &Stmt{Stmt: stmt, metrics: c.metrics}, nil
}

func (c *Conn) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	defer recordQuery(c.metrics, recordKindWrite)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	res, err := c.Conn.ExecContext(ctx, query, args...)
	if err != nil {
		return res, repoerr(err)
	}

	return res, nil
}

func (c *Conn) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	defer recordQuery(c.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	_rows, err := c.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repoerr(err)
	}

	rows := &Rows{Rows: _rows, metrics: c.metrics}

	rows.record()

	return rows, nil
}

func (c *Conn) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	defer recordQuery(c.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return &Row{err: err}
		}
	}

	_row := c.Conn.QueryRowContext(ctx, query, args...)
	row := &Row{Row: _row, metrics: c.metrics}

	row.record()

	return row
}

type DB struct {
	*sql.DB
	metrics *expvar.Map
}

func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return &Conn{Conn: conn, metrics: db.metrics}, nil
}

func (db *DB) Begin() (*Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	if db.metrics != nil {
		db.metrics.Add("totalTransactionsBegun", 1)
	}

	_tx := &Tx{Tx: tx, ctx: ctx, now: time.Now(), metrics: db.metrics}

	go _tx.awaitDone()

	return _tx, nil
}

// BeginImmediateTx starts an immediate transaction with "BEGIN IMMEDIATE".
//
// This is a workaround for Go's database/sql package not providing a way to set
// the transaction type per connection.
//
// References:
// - https://github.com/golang/go/issues/19981
// - https://github.com/mattn/go-sqlite3/issues/400
func (db *DB) BeginImmediateTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// The returned transaction is a connection from a connection pool, so we
	// can rollback the (by default) "DEFERRED" transaction and start a new
	// "IMMEDIATE" one
	if _, err := tx.ExecContext(ctx, "ROLLBACK; BEGIN IMMEDIATE"); err != nil {
		return nil, err
	}

	if db.metrics != nil {
		db.metrics.Add("totalTransactionsBegun", 1)
	}

	return tx, nil
}

// BeginExclusiveTx starts an exclusive transaction with "BEGIN EXCLUSIVE".
//
// This is a workaround for Go's database/sql package not providing a way to set
// the transaction type per connection.
//
// References:
// - https://github.com/golang/go/issues/19981
// - https://github.com/mattn/go-sqlite3/issues/400
func (db *DB) BeginExclusiveTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// The returned transaction is a connection from a connection pool, so we
	// can rollback the (by default) "DEFERRED" transaction and start a new
	// "EXCLUSIVE" one
	if _, err := tx.ExecContext(ctx, "ROLLBACK; BEGIN EXCLUSIVE"); err != nil {
		return nil, err
	}

	if db.metrics != nil {
		db.metrics.Add("totalTransactionsBegun", 1)
	}

	return tx, nil
}

func (db *DB) Prepare(query string) (*Stmt, error) {
	return db.PrepareContext(context.Background(), query)
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := db.DB.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return &Stmt{Stmt: stmt, metrics: db.metrics}, nil
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	defer recordQuery(db.metrics, recordKindWrite)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	res, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return res, repoerr(err)
	}

	return res, nil
}

func (db *DB) Query(query string, args ...any) (*Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	defer recordQuery(db.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	_rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repoerr(err)
	}

	rows := &Rows{Rows: _rows, metrics: db.metrics}

	rows.record()

	return rows, nil
}

func (db *DB) QueryRow(query string, args ...any) *Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	defer recordQuery(db.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return &Row{err: err}
		}
	}

	_row := db.DB.QueryRowContext(ctx, query, args...)
	row := &Row{Row: _row, metrics: db.metrics}

	row.record()

	return row
}

type Stmt struct {
	*sql.Stmt
	metrics *expvar.Map
}

func (stmt *Stmt) Exec(args ...any) (sql.Result, error) {
	return stmt.ExecContext(context.Background(), args...)
}

func (stmt *Stmt) ExecContext(ctx context.Context, args ...any) (sql.Result, error) {
	defer recordQuery(stmt.metrics, recordKindWrite)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	res, err := stmt.Stmt.ExecContext(ctx, args...)
	if err != nil {
		return res, repoerr(err)
	}

	return res, nil
}

func (stmt *Stmt) Query(args ...any) (*Rows, error) {
	return stmt.QueryContext(context.Background(), args...)
}

func (stmt *Stmt) QueryContext(ctx context.Context, args ...any) (*Rows, error) {
	defer recordQuery(stmt.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	_rows, err := stmt.Stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, repoerr(err)
	}

	rows := &Rows{Rows: _rows, metrics: stmt.metrics}

	rows.record()

	return rows, nil
}

func (stmt *Stmt) QueryRow(args ...any) *Row {
	return stmt.QueryRowContext(context.Background(), args...)
}

func (stmt *Stmt) QueryRowContext(ctx context.Context, args ...any) *Row {
	defer recordQuery(stmt.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return &Row{err: err}
		}
	}

	_row := stmt.Stmt.QueryRowContext(ctx, args...)
	row := &Row{Row: _row, metrics: stmt.metrics}

	row.record()

	return row
}

type Tx struct {
	*sql.Tx
	ctx        context.Context
	now        time.Time
	metrics    *expvar.Map
	recorded   atomic.Bool
	maybeWrite bool
}

func (tx *Tx) awaitDone() {
	if tx.metrics == nil {
		return
	}

	<-tx.ctx.Done()

	tx.Rollback()

	if tx.recorded.CompareAndSwap(false, true) {
		tx.metrics.Add("totalTransactionsCancelled", 1)
	}
}

func (tx *Tx) Commit() error {
	err := tx.Tx.Commit()
	if err == nil && tx.metrics != nil && tx.recorded.CompareAndSwap(false, true) {
		tx.metrics.Add("totalTransactionsCommitted", 1)
	}

	return err
}

func (tx *Tx) Rollback() error {
	err := tx.Tx.Rollback()
	if err == nil && tx.metrics != nil && tx.recorded.CompareAndSwap(false, true) {
		// When using Go's SQL package queries that can modify the
		// database should be called using the Exec* methods
		//
		// Because of this, and because a transaction of only SELECTs
		// doesn't need to be committed, we assume that if an Exec*
		// method wasn't called then it counts as a read-only transaction
		// where a call to rollback was used instead of commit for brevity in code
		//
		// In those cases we choose to increment the committed metrics
		// rather than the rolled back one since it's not likely actually
		// rolling anything back
		if tx.maybeWrite {
			tx.metrics.Add("totalTransactionsRolledBack", 1)
		} else {
			tx.metrics.Add("totalTransactionsCommitted", 1)
		}
	}

	return err
}

func (tx *Tx) Prepare(query string) (*Stmt, error) {
	return tx.PrepareContext(context.Background(), query)
}

func (tx *Tx) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := tx.Tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return &Stmt{Stmt: stmt, metrics: tx.metrics}, nil
}

func (tx *Tx) Stmt(stmt *Stmt) *Stmt {
	return tx.StmtContext(context.Background(), stmt)
}

func (tx *Tx) StmtContext(ctx context.Context, stmt *Stmt) *Stmt {
	return &Stmt{Stmt: tx.Tx.StmtContext(ctx, stmt.Stmt), metrics: tx.metrics}
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(context.Background(), query, args...)
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	recordKind := recordKindRead

	// A query that rolls back and begins an exclusive/immediate transaction is used
	// as a workaround for starting exclusive/immediate transactions with Go's SQL
	// interface, which doesn't allow setting the transaction type through a normal
	// method call
	//
	// In these cases we don't want to record the exec call because it will be used to
	// determine whether a transaction possibly modified the database
	if query != "ROLLBACK; BEGIN EXCLUSIVE" && query != "ROLLBACK; BEGIN IMMEDIATE" {
		recordKind = recordKindWrite
		tx.maybeWrite = true
	}

	defer recordQuery(tx.metrics, recordKind)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	res, err := tx.Tx.ExecContext(ctx, query, args...)
	if err != nil {
		return res, repoerr(err)
	}

	return res, nil
}

func (tx *Tx) Query(query string, args ...any) (*Rows, error) {
	return tx.QueryContext(context.Background(), query, args...)
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	defer recordQuery(tx.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
	}

	_rows, err := tx.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repoerr(err)
	}

	rows := &Rows{Rows: _rows, metrics: tx.metrics}

	rows.record()

	return rows, nil
}

func (tx *Tx) QueryRow(query string, args ...any) *Row {
	return tx.QueryRowContext(context.Background(), query, args...)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	defer recordQuery(tx.metrics, recordKindRead)()

	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return &Row{err: err}
		}
	}

	_row := tx.Tx.QueryRowContext(ctx, query, args...)
	row := &Row{Row: _row, metrics: tx.metrics}

	row.record()

	return row
}

type Row struct {
	*sql.Row
	err      error
	openedAt time.Time
	metrics  *expvar.Map
}

func (r *Row) record() {
	if r.metrics == nil || r.Err() != nil {
		return
	}

	r.openedAt = time.Now()

	r.metrics.Add("totalRowsOpened", 1)
}

func (r *Row) Err() error {
	if r.err != nil {
		return r.err
	}

	return repoerr(r.Row.Err())
}

func (r *Row) Scan(dst ...any) error {
	if err := r.Err(); err != nil {
		return err
	}

	if r.metrics != nil {
		// We defer the recording of rows closed and rows time here because we want to make
		// sure it runs after the scan call whether there was an error or not
		// This is because the scan call is where the rows are actually closed
		defer func() {
			r.metrics.Add("totalRowsClosed", 1)
			r.metrics.Add("totalRowsTime", time.Since(r.openedAt).Nanoseconds())
		}()
	}

	if err := repoerr(r.Row.Scan(dst...)); err != nil {
		return err
	}

	return nil
}

type Rows struct {
	*sql.Rows
	openedAt time.Time
	metrics  *expvar.Map
}

func (rs *Rows) record() {
	if rs.metrics == nil {
		return
	}

	rs.openedAt = time.Now()

	rs.metrics.Add("totalRowsOpened", 1)
}

func (rs *Rows) Close() error {
	if err := repoerr(rs.Rows.Close()); err != nil {
		return err
	}

	if rs.metrics != nil {
		rs.metrics.Add("totalRowsClosed", 1)
		rs.metrics.Add("totalRowsTime", time.Since(rs.openedAt).Nanoseconds())
	}

	return nil
}

func (rs *Rows) Err() error {
	return repoerr(rs.Rows.Err())
}

func (rs *Rows) Next() bool {
	return rs.Rows.Next()
}

func (rs *Rows) Scan(dst ...any) error {
	return repoerr(rs.Rows.Scan(dst...))
}

func newSorts(sorts []string, keysToCols map[string]string) []string {
	if len(sorts) == 0 || len(keysToCols) == 0 {
		return nil
	}

	var results []string
	for _, pair := range sorts {
		key, dir, _ := strings.Cut(pair, ".")

		key = strings.TrimSpace(key)

		dir = strings.TrimSpace(dir)
		dir = strings.ToLower(dir)

		col := keysToCols[key]
		if col == "" {
			continue
		}

		switch dir {
		case "asc":
			results = append(results, col+" ASC")

		case "desc":
			results = append(results, col+" DESC")
		}
	}

	return results
}
