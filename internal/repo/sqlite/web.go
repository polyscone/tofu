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
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/repo"
)

const (
	webTokenKindActivation    = "activation"
	webTokenKindResetPassword = "reset_password"
)

type WebStore struct {
	db *DB
}

func NewWebStore(ctx context.Context, db *sql.DB, sessionLifespan time.Duration) (*WebStore, error) {
	migrations, err := fs.Sub(migrations, "migrations/web")
	if err != nil {
		return nil, fmt.Errorf("initialise web migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db, "web", migrations); err != nil {
		return nil, fmt.Errorf("migrate web: %w", err)
	}

	s := WebStore{db: newDB(db)}

	// Background goroutine to clean up expired sessions
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(sessionLifespan) {
			if err := s.DestroyExpiredSessions(ctx, sessionLifespan); err != nil {
				logger.PrintErrorf("web store: destroy expired sessions: %w", err)
			}
		}
	})

	// Background goroutine to clean up expired tokens
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			if err := s.DeleteExpiredTokens(ctx); err != nil {
				logger.PrintErrorf("web store: delete expired tokens: %w", err)
			}
		}
	})

	return &s, nil
}

func (s *WebStore) FindSessionDataByID(ctx context.Context, id string) (session.Data, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return s.findSessionDataByID(ctx, tx, id)
}

func (s *WebStore) SaveSession(ctx context.Context, sess session.Session) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.upsertSession(ctx, tx, sess); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (s *WebStore) DestroySession(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.destroySession(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (s *WebStore) DestroyExpiredSessions(ctx context.Context, lifespan time.Duration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.destroyExpiredSessions(ctx, tx, lifespan); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (s *WebStore) FindActivationTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return s.findToken(ctx, tx, token, webTokenKindActivation)
}

func (s *WebStore) FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return s.findToken(ctx, tx, token, webTokenKindResetPassword)
}

func (s *WebStore) AddActivationToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	token, err := s.createToken(ctx, tx, email, ttl, webTokenKindActivation)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (s *WebStore) AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	token, err := s.createToken(ctx, tx, email, ttl, webTokenKindResetPassword)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx commit: %w", err)
	}

	return token, nil
}

func (s *WebStore) ConsumeActivationToken(ctx context.Context, token string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.consumeToken(ctx, tx, token, webTokenKindActivation); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (s *WebStore) ConsumeResetPasswordToken(ctx context.Context, token string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.consumeToken(ctx, tx, token, webTokenKindResetPassword); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (s *WebStore) DeleteExpiredTokens(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.deleteExpiredTokens(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (s *WebStore) findSessionDataByID(ctx context.Context, tx *Tx, id string) (session.Data, error) {
	var data []byte

	err := tx.QueryRowContext(ctx, `
		SELECT data
		FROM web__sessions
		WHERE id = :id
	`,
		sql.Named("id", id),
	).Scan(&data)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
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

func (s *WebStore) upsertSession(ctx context.Context, tx *Tx, sess session.Session) error {
	b, err := json.Marshal(sess.Data)
	if err != nil {
		return fmt.Errorf("marshal session data JSON: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO web__sessions (
			id,
			data,
			created_at,
			updated_at
		) VALUES (
			:id,
			:data,
			:created_at,
			:updated_at
		)
		ON CONFLICT DO
			UPDATE SET
				data = :data,
				updated_at = :updated_at
	`,
		sql.Named("id", sess.ID),
		sql.Named("data", b),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)

	return err
}

func (s *WebStore) destroySession(ctx context.Context, tx *Tx, id string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE id = :id
	`,
		sql.Named("id", id),
	)

	return err
}

func (s *WebStore) destroyExpiredSessions(ctx context.Context, tx *Tx, lifespan time.Duration) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE updated_at <= :expires_at
	`,
		sql.Named("expires_at", Time(tx.now.Add(-lifespan).UTC())),
	)

	return err
}

func (s *WebStore) findToken(ctx context.Context, tx *Tx, token, kind string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var email string

	err := tx.QueryRowContext(ctx, `
		SELECT email
		FROM web__tokens
		WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(tx.now.UTC())),
	).Scan(&email)

	return email, err
}

func (s *WebStore) consumeToken(ctx context.Context, tx *Tx, token, kind string) error {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	res, err := tx.ExecContext(ctx, `
		DELETE FROM web__tokens	WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(tx.now.UTC())),
	)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if affected == 0 {
		return repo.ErrNotFound
	}

	return nil
}

func (s *WebStore) createToken(ctx context.Context, tx *Tx, email string, ttl time.Duration, kind string) (string, error) {
	b := make([]byte, 16)
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
		INSERT INTO web__tokens (
			hash,
			email,
			kind,
			expires_at,
			created_at
		) VALUES (
			:hash,
			:email,
			:kind,
			:expires_at,
			:created_at
		)
	`,
		sql.Named("hash", hash),
		sql.Named("email", email),
		sql.Named("kind", kind),
		sql.Named("expires_at", expiresAt),
		sql.Named("created_at", Time(tx.now.UTC())),
	)

	return string(token), err
}

func (s *WebStore) deleteExpiredTokens(ctx context.Context, tx *Tx) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__tokens
		WHERE expires_at <= :expires_at
	`,
		sql.Named("expires_at", Time(tx.now.UTC())),
	)

	return err
}
