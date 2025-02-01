package sqlite

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/internal/session"
)

const (
	webTokenKindEmailVerification = "email_verification"
	webTokenKindResetPassword     = "reset_password"
	webTokenKindSignInMagicLink   = "sign_in_magic_link"
	webTokenKindTOTPResetVerify   = "totp_reset_verify"
	webTokenKindTOTPReset         = "totp_reset"
)

type WebRepo struct {
	db         *DB
	sessionTTL time.Duration
}

func NewWebRepo(ctx context.Context, db *DB, sessionTTL time.Duration) (*WebRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/web")
	if err != nil {
		return nil, fmt.Errorf("initialize web migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db, "web", migrations); err != nil {
		return nil, fmt.Errorf("migrate web: %w", err)
	}

	r := WebRepo{
		db:         db,
		sessionTTL: sessionTTL,
	}

	// Background goroutine to clean up expired sessions
	background.GoUnawaited(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			validWindowStart := time.Now().Add(-r.sessionTTL).UTC()
			if err := r.DestroyExpiredSessions(ctx, validWindowStart); err != nil {
				slog.Error("web repo: destroy expired sessions", "error", err)
			}
		}
	})

	// Background goroutine to clean up expired tokens
	background.GoUnawaited(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			now := time.Now().UTC()
			if err := r.DeleteExpiredTokens(ctx, now); err != nil {
				slog.Error("web repo: delete expired tokens", "error", err)
			}
		}
	})

	// Background goroutine to clean up old domain events
	background.GoUnawaited(func() {
		ctx := context.Background()

		for range time.Tick(1 * time.Hour) {
			const ttl = 60 * 24 * time.Hour
			validWindowStart := time.Now().Add(-ttl).UTC()
			if err := r.ClearOldDomainEvents(ctx, validWindowStart); err != nil {
				slog.Error("web repo: clear old domain events", "error", err)
			}
		}
	})

	return &r, nil
}

func (r *WebRepo) FindSessionDataByID(ctx context.Context, id string) (session.Data, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findSessionDataByID(ctx, tx, id)
}

func (r *WebRepo) SaveSession(ctx context.Context, sess session.Session) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.upsertSession(ctx, tx, sess); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) DestroySession(ctx context.Context, id string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.destroySession(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) CountExpiredSessions(ctx context.Context, validWindowStart time.Time) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	total, err := r.countExpiredSessions(ctx, tx, validWindowStart)

	return total, err
}

func (r *WebRepo) DestroyExpiredSessions(ctx context.Context, validWindowStart time.Time) error {
	total, err := r.CountExpiredSessions(ctx, validWindowStart)
	if err != nil {
		return fmt.Errorf("count expired sessions: %w", err)
	}
	if total == 0 {
		return nil
	}

	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.destroyExpiredSessions(ctx, tx, validWindowStart); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) FindEmailVerificationTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findToken(ctx, tx, token, webTokenKindEmailVerification)
}

func (r *WebRepo) AddEmailVerificationToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	token, err := r.createToken(ctx, tx, email, ttl, webTokenKindEmailVerification)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (r *WebRepo) ConsumeEmailVerificationToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	email, err := r.consumeToken(ctx, tx, token, webTokenKindEmailVerification)
	if err != nil {
		return err
	}

	if err := r.deleteTokensByKind(ctx, tx, email, webTokenKindEmailVerification); err != nil {
		return fmt.Errorf("delete tokens for email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findToken(ctx, tx, token, webTokenKindResetPassword)
}

func (r *WebRepo) AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	token, err := r.createToken(ctx, tx, email, ttl, webTokenKindResetPassword)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (r *WebRepo) ConsumeResetPasswordToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	email, err := r.consumeToken(ctx, tx, token, webTokenKindResetPassword)
	if err != nil {
		return err
	}

	if err := r.deleteTokensByKind(ctx, tx, email, webTokenKindResetPassword); err != nil {
		return fmt.Errorf("delete tokens for email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) FindSignInMagicLinkTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findToken(ctx, tx, token, webTokenKindSignInMagicLink)
}

func (r *WebRepo) AddSignInMagicLinkToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.deleteTokensByKind(ctx, tx, email, webTokenKindSignInMagicLink); err != nil {
		return "", fmt.Errorf("delete tokens for email: %w", err)
	}

	token, err := r.createToken(ctx, tx, email, ttl, webTokenKindSignInMagicLink)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (r *WebRepo) ConsumeSignInMagicLinkToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	email, err := r.consumeToken(ctx, tx, token, webTokenKindSignInMagicLink)
	if err != nil {
		return err
	}

	if err := r.deleteTokensByKind(ctx, tx, email, webTokenKindSignInMagicLink); err != nil {
		return fmt.Errorf("delete tokens for email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) FindTOTPResetVerifyTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findToken(ctx, tx, token, webTokenKindTOTPResetVerify)
}

func (r *WebRepo) AddTOTPResetVerifyToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	token, err := r.createToken(ctx, tx, email, ttl, webTokenKindTOTPResetVerify)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (r *WebRepo) ConsumeTOTPResetVerifyToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	email, err := r.consumeToken(ctx, tx, token, webTokenKindTOTPResetVerify)
	if err != nil {
		return err
	}

	if err := r.deleteTokensByKind(ctx, tx, email, webTokenKindTOTPResetVerify); err != nil {
		return fmt.Errorf("delete tokens for email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) FindResetTOTPTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findToken(ctx, tx, token, webTokenKindTOTPReset)
}

func (r *WebRepo) AddResetTOTPToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	token, err := r.createToken(ctx, tx, email, ttl, webTokenKindTOTPReset)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (r *WebRepo) ConsumeResetTOTPToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	email, err := r.consumeToken(ctx, tx, token, webTokenKindTOTPReset)
	if err != nil {
		return err
	}

	if err := r.deleteTokensByKind(ctx, tx, email, webTokenKindTOTPReset); err != nil {
		return fmt.Errorf("delete tokens for email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) CountExpiredTokens(ctx context.Context, now time.Time) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	total, err := r.countExpiredTokens(ctx, tx, now)

	return total, err
}

func (r *WebRepo) DeleteExpiredTokens(ctx context.Context, now time.Time) error {
	total, err := r.CountExpiredTokens(ctx, now)
	if err != nil {
		return fmt.Errorf("count expired tokens: %w", err)
	}
	if total == 0 {
		return nil
	}

	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.deleteExpiredTokens(ctx, tx, now); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) LogDomainEvent(ctx context.Context, kind, name, data string) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.createDomainEvent(ctx, tx, kind, name, data); err != nil {
		return fmt.Errorf("create domain event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) ClearOldDomainEvents(ctx context.Context, validWindowStart time.Time) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.clearOldDomainEvents(ctx, tx, validWindowStart); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *WebRepo) findSessionDataByID(ctx context.Context, tx *Tx, id string) (session.Data, error) {
	var data []byte
	err := tx.QueryRowContext(ctx, `
		select data
		from web__sessions
		where
			id = :id and
			updated_at > :valid_window_start
	`,
		sql.Named("id", id),
		sql.Named("valid_window_start", Time(tx.now.Add(-r.sessionTTL).UTC())),
	).Scan(&data)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return nil, session.ErrNotFound
		}

		return nil, err
	}

	d := json.NewDecoder(bytes.NewReader(data))

	d.UseNumber()

	var res session.Data
	if err := d.Decode(&res); err != nil {
		return nil, fmt.Errorf("decode session data JSON: %w", err)
	}

	return res, nil
}

func (r *WebRepo) upsertSession(ctx context.Context, tx *Tx, sess session.Session) error {
	b, err := json.Marshal(sess.Data)
	if err != nil {
		return fmt.Errorf("marshal session data JSON: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		insert into web__sessions (
			id,
			data,
			created_at,
			updated_at
		) values (
			:id,
			:data,
			:created_at,
			:updated_at
		)
		on conflict (id) do
			update set
				data = excluded.data,
				updated_at = :updated_at
	`,
		sql.Named("id", sess.ID),
		sql.Named("data", b),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)

	return err
}

func (r *WebRepo) destroySession(ctx context.Context, tx *Tx, id string) error {
	_, err := tx.ExecContext(ctx, `
		delete from web__sessions
		where id = :id
	`,
		sql.Named("id", id),
	)

	return err
}

func (r *WebRepo) countExpiredSessions(ctx context.Context, tx *Tx, validWindowStart time.Time) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx, `
		select count(*) from web__sessions
		where updated_at <= :valid_window_start
	`,
		sql.Named("valid_window_start", Time(validWindowStart)),
	).Scan(&count)

	return count, err
}

func (r *WebRepo) destroyExpiredSessions(ctx context.Context, tx *Tx, validWindowStart time.Time) error {
	_, err := tx.ExecContext(ctx, `
		delete from web__sessions
		where updated_at <= :valid_window_start
	`,
		sql.Named("valid_window_start", Time(validWindowStart)),
	)

	return err
}

func (r *WebRepo) findToken(ctx context.Context, tx *Tx, token, kind string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var value string
	err := tx.QueryRowContext(ctx, `
		select value
		from web__tokens
		where
			hash = :hash and
			kind = :kind and
			expires_at > :expires_at
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(tx.now.UTC())),
	).Scan(&value)

	return value, err
}

func (r *WebRepo) consumeToken(ctx context.Context, tx *Tx, token, kind string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var value string
	err := tx.QueryRowContext(ctx, `
		delete from web__tokens
		where
			hash = :hash and
			kind = :kind and
			expires_at > :expires_at
		returning value
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(tx.now.UTC())),
	).Scan(&value)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", app.ErrNotFound
	}

	return value, nil
}

func (r *WebRepo) createToken(ctx context.Context, tx *Tx, value string, ttl time.Duration, kind string) (string, error) {
	b := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	b32 := base32.StdEncoding.WithPadding(base32.NoPadding)
	token := make([]byte, b32.EncodedLen(len(b)))
	b32.Encode(token, b)

	sum := sha256.Sum256(token)
	hash := sum[:]

	expiresAt := Time(tx.now.Add(ttl).UTC())

	_, err := tx.ExecContext(ctx, `
		insert into web__tokens (
			hash,
			value,
			kind,
			expires_at,
			created_at
		) values (
			:hash,
			:value,
			:kind,
			:expires_at,
			:created_at
		)
	`,
		sql.Named("hash", hash),
		sql.Named("value", value),
		sql.Named("kind", kind),
		sql.Named("expires_at", expiresAt),
		sql.Named("created_at", Time(tx.now.UTC())),
	)

	return string(token), err
}

func (r *WebRepo) deleteTokensByKind(ctx context.Context, tx *Tx, value string, kind Kind) error {
	_, err := tx.ExecContext(ctx, `
		delete from web__tokens
		where
			value = :value and
			kind = :kind
	`,
		sql.Named("value", value),
		sql.Named("kind", kind),
	)

	return err
}

func (r *WebRepo) countExpiredTokens(ctx context.Context, tx *Tx, now time.Time) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx, `
		select count(*) from web__tokens
		where expires_at <= :now
	`,
		sql.Named("now", Time(now)),
	).Scan(&count)

	return count, err
}

func (r *WebRepo) deleteExpiredTokens(ctx context.Context, tx *Tx, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
		delete from web__tokens
		where expires_at <= :now
	`,
		sql.Named("now", Time(now)),
	)

	return err
}

func (r *WebRepo) createDomainEvent(ctx context.Context, tx *Tx, kind, name, data string) error {
	_, err := tx.ExecContext(ctx, `
		insert into web__domain_events (
			kind,
			name,
			data,
			created_at
		) values (
			:kind,
			:name,
			:data,
			:created_at
		)
	`,
		sql.Named("kind", kind),
		sql.Named("name", name),
		sql.Named("data", data),
		sql.Named("created_at", Time(tx.now.UTC())),
	)

	return err
}

func (r *WebRepo) clearOldDomainEvents(ctx context.Context, tx *Tx, validWindowStart time.Time) error {
	_, err := tx.ExecContext(ctx, `
		delete from web__domain_events
		where created_at <= :valid_window_start
	`,
		sql.Named("valid_window_start", Time(validWindowStart)),
	)

	return err
}
