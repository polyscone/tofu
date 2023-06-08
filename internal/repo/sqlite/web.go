package sqlite

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/fs"
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
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
		return nil, errors.Tracef(err)
	}

	if err := migrateFS(ctx, db, "web", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	s := WebStore{db: newDB(db)}

	// Background goroutine to clean up expired sessions
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(sessionLifespan) {
			if err := s.DestroyExpiredSessions(ctx, sessionLifespan); err != nil {
				logger.PrintError(errors.Tracef(err))
			}
		}
	})

	// Background goroutine to clean up expired tokens
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			if err := s.DeleteExpiredTokens(ctx); err != nil {
				logger.PrintError(errors.Tracef(err))
			}
		}
	})

	return &s, nil
}

func (s *WebStore) FindSessionDataByID(ctx context.Context, id string) (session.Data, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	session, err := s.findSessionDataByID(ctx, tx, id)

	return session, errors.Tracef(err)
}

func (s *WebStore) SaveSession(ctx context.Context, sess session.Session) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.saveSession(ctx, tx, sess)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *WebStore) DestroySession(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.destroySession(ctx, tx, id)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *WebStore) DestroyExpiredSessions(ctx context.Context, lifespan time.Duration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.destroyExpiredSessions(ctx, tx, lifespan)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *WebStore) FindActivationTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err = s.findToken(ctx, tx, token, webTokenKindActivation)

	return token, errors.Tracef(err)
}

func (s *WebStore) FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err = s.findToken(ctx, tx, token, webTokenKindResetPassword)

	return token, errors.Tracef(err)
}

func (s *WebStore) AddActivationToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err := s.addToken(ctx, tx, email, ttl, webTokenKindActivation)
	if err != nil {
		return "", errors.Tracef(err)
	}

	return token, errors.Tracef(tx.Commit())
}

func (s *WebStore) AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err := s.addToken(ctx, tx, email, ttl, webTokenKindResetPassword)
	if err != nil {
		return "", errors.Tracef(err)
	}

	return token, errors.Tracef(tx.Commit())
}

func (s *WebStore) ConsumeActivationToken(ctx context.Context, token string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.consumeToken(ctx, tx, token, webTokenKindActivation)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *WebStore) ConsumeResetPasswordToken(ctx context.Context, token string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.consumeToken(ctx, tx, token, webTokenKindResetPassword)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *WebStore) DeleteExpiredTokens(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.deleteExpiredTokens(ctx, tx)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
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
			err = errors.Tracef(session.ErrNotFound, err)
		}

		return nil, errors.Tracef(err)
	}

	d := json.NewDecoder(bytes.NewReader(data))

	d.UseNumber()

	var res session.Data
	err = d.Decode(&res)

	return res, errors.Tracef(err)
}

func (s *WebStore) saveSession(ctx context.Context, tx *Tx, sess session.Session) error {
	b, err := json.Marshal(sess.Data)
	if err != nil {
		return errors.Tracef(err)
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
		) ON CONFLICT DO UPDATE SET
			data = :data,
			updated_at = :updated_at
	`,
		sql.Named("id", sess.ID),
		sql.Named("data", b),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)

	return errors.Tracef(err)
}

func (s *WebStore) destroySession(ctx context.Context, tx *Tx, id string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE id = :id
	`,
		sql.Named("id", id),
	)

	return errors.Tracef(err)
}

func (s *WebStore) destroyExpiredSessions(ctx context.Context, tx *Tx, lifespan time.Duration) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE updated_at <= :expires_at
	`,
		sql.Named("expires_at", Time(tx.now.Add(-lifespan).UTC())),
	)

	return errors.Tracef(err)
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

	return email, errors.Tracef(err)
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
		return errors.Tracef(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Tracef(err)
	}
	if affected == 0 {
		return errors.Tracef(repo.ErrNotFound)
	}

	return nil
}

func (s *WebStore) addToken(ctx context.Context, tx *Tx, email string, ttl time.Duration, kind string) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Tracef(err)
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
	if err != nil {
		return "", errors.Tracef(err)
	}

	return string(token), nil
}

func (s *WebStore) deleteExpiredTokens(ctx context.Context, tx *Tx) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__tokens
		WHERE expires_at <= :expires_at
	`,
		sql.Named("expires_at", Time(tx.now.UTC())),
	)

	return errors.Tracef(err)
}
