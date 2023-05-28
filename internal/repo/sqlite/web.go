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

type WebRepo struct {
	db *DB
}

func NewWebRepo(ctx context.Context, db *sql.DB, sessionLifespan time.Duration) (*WebRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/web")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := migrateFS(ctx, db, "web", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	r := WebRepo{db: newDB(db)}

	// Background goroutine to clean up expired sessions
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(sessionLifespan) {
			if err := r.DestroyExpiredSessions(ctx, sessionLifespan); err != nil {
				logger.PrintError(err)
			}
		}
	})

	// Background goroutine to clean up expired tokens
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			if err := r.DeleteExpiredTokens(ctx); err != nil {
				logger.PrintError(err)
			}
		}
	})

	return &r, nil
}

func (r *WebRepo) FindSessionDataByID(ctx context.Context, id string) (session.Data, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	session, err := r.findSessionDataByID(ctx, tx, id)

	return session, errors.Tracef(err)
}

func (r *WebRepo) SaveSession(ctx context.Context, s session.Session) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = r.saveSession(ctx, tx, s)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *WebRepo) DestroySession(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = r.destroySession(ctx, tx, id)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *WebRepo) DestroyExpiredSessions(ctx context.Context, lifespan time.Duration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = r.destroyExpiredSessions(ctx, tx, lifespan)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *WebRepo) FindActivationTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err = r.findToken(ctx, tx, token, webTokenKindActivation)

	return token, errors.Tracef(err)
}

func (r *WebRepo) FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err = r.findToken(ctx, tx, token, webTokenKindResetPassword)

	return token, errors.Tracef(err)
}

func (r *WebRepo) AddActivationToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err := r.addToken(ctx, tx, email, ttl, webTokenKindActivation)
	if err != nil {
		return "", errors.Tracef(err)
	}

	return token, errors.Tracef(tx.Commit())
}

func (r *WebRepo) AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	token, err := r.addToken(ctx, tx, email, ttl, webTokenKindResetPassword)
	if err != nil {
		return "", errors.Tracef(err)
	}

	return token, errors.Tracef(tx.Commit())
}

func (r *WebRepo) ConsumeActivationToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = r.consumeToken(ctx, tx, token, webTokenKindActivation)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *WebRepo) ConsumeResetPasswordToken(ctx context.Context, token string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = r.consumeToken(ctx, tx, token, webTokenKindResetPassword)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *WebRepo) DeleteExpiredTokens(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = r.deleteExpiredTokens(ctx, tx)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *WebRepo) findSessionDataByID(ctx context.Context, tx *Tx, id string) (session.Data, error) {
	var data []byte

	err := tx.QueryRowContext(ctx, `
		SELECT data
		FROM web__sessions
		WHERE id = :id;
	`,
		sql.Named("id", id),
	).Scan(&data)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			err = errors.Tracef(err, session.ErrNotFound)
		}

		return nil, errors.Tracef(err)
	}

	d := json.NewDecoder(bytes.NewReader(data))

	d.UseNumber()

	var res session.Data
	err = d.Decode(&res)

	return res, errors.Tracef(err)
}

func (r *WebRepo) saveSession(ctx context.Context, tx *Tx, s session.Session) error {
	b, err := json.Marshal(s.Data)
	if err != nil {
		return errors.Tracef(err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO web__sessions (
			id,
			data,
			updated_at
		) VALUES (
			:id,
			:data,
			:updated_at
		);
	`,
		sql.Named("id", s.ID),
		sql.Named("data", b),
		sql.Named("updated_at", Time(time.Now().UTC())),
	)

	return errors.Tracef(err)
}

func (r *WebRepo) destroySession(ctx context.Context, tx *Tx, id string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE id = :id;
	`,
		sql.Named("id", id),
	)

	return errors.Tracef(err)
}

func (r *WebRepo) destroyExpiredSessions(ctx context.Context, tx *Tx, lifespan time.Duration) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE updated_at <= :expires_at;
	`,
		sql.Named("expires_at", Time(time.Now().Add(-lifespan).UTC())),
	)

	return errors.Tracef(err)
}

func (r *WebRepo) findToken(ctx context.Context, tx *Tx, token, kind string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var email string

	err := tx.QueryRowContext(ctx, `
		SELECT email
		FROM web__tokens
		WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at;
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(time.Now().UTC())),
	).Scan(&email)

	return email, errors.Tracef(err)
}

func (r *WebRepo) consumeToken(ctx context.Context, tx *Tx, token, kind string) error {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	res, err := tx.ExecContext(ctx, `
		DELETE FROM web__tokens	WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at;
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(time.Now().UTC())),
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

func (r *WebRepo) addToken(ctx context.Context, tx *Tx, email string, ttl time.Duration, kind string) (string, error) {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__tokens
		WHERE
			email = :email AND
			kind = :kind;
	`,
		sql.Named("email", email),
		sql.Named("kind", kind),
	)
	if err != nil {
		return "", errors.Tracef(err)
	}

	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Tracef(err)
	}

	b32 := base32.StdEncoding.WithPadding(base32.NoPadding)
	token := make([]byte, b32.EncodedLen(len(b)))
	b32.Encode(token, b)

	sum := sha256.Sum256(token)
	hash := sum[:]

	expiresAt := Time(time.Now().Add(ttl).UTC())

	_, err = tx.ExecContext(ctx, `
		INSERT INTO web__tokens (
			hash,
			email,
			kind,
			expires_at
		) VALUES (
			:hash,
			:email,
			:kind,
			:expires_at
		);
	`,
		sql.Named("hash", hash),
		sql.Named("email", email),
		sql.Named("kind", kind),
		sql.Named("expires_at", expiresAt),
	)
	if err != nil {
		return "", errors.Tracef(err)
	}

	return string(token), nil
}

func (r *WebRepo) deleteExpiredTokens(ctx context.Context, tx *Tx) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM web__tokens
		WHERE expires_at <= :expires_at;
	`,
		sql.Named("expires_at", Time(time.Now().UTC())),
	)

	return errors.Tracef(err)
}
