package sqlite

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
)

type SystemStore struct {
	db *DB
}

func NewSystemStore(ctx context.Context, db *sql.DB) (*SystemStore, error) {
	migrations, err := fs.Sub(migrations, "migrations/system")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := migrateFS(ctx, db, "system", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	s := SystemStore{db: newDB(db)}

	return &s, nil
}

func (s *SystemStore) FindConfig(ctx context.Context) (*system.Config, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	config, err := s.findConfig(ctx, tx)

	return config, errors.Tracef(err)
}

func (s *SystemStore) SaveConfig(ctx context.Context, config *system.Config) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	err = s.upsertConfig(ctx, tx, config)
	if err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *SystemStore) findConfig(ctx context.Context, tx *Tx) (*system.Config, error) {
	var config system.Config

	err := tx.QueryRowContext(ctx, `
		SELECT
			system_email,
			twilio_sid,
			twilio_token,
			twilio_from_tel
		FROM system__config
	`).Scan(
		&config.SystemEmail,
		&config.TwilioSID,
		&config.TwilioToken,
		&config.TwilioFromTel,
	)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, errors.Tracef(err)
	}

	config.RequiresSetup = errors.Is(err, repo.ErrNotFound)

	return &config, nil
}

func (s *SystemStore) upsertConfig(ctx context.Context, tx *Tx, config *system.Config) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO system__config (
			id,
			system_email,
			twilio_sid,
			twilio_token,
			twilio_from_tel,
			created_at
		) VALUES (
			:id,
			:system_email,
			:twilio_sid,
			:twilio_token,
			:twilio_from_tel,
			:created_at
		)
		ON CONFLICT DO
			UPDATE SET
				system_email = :system_email,
				twilio_sid = :twilio_sid,
				twilio_token = :twilio_token,
				twilio_from_tel = :twilio_from_tel,
				updated_at = :updated_at
	`,
		sql.Named("id", 1),
		sql.Named("system_email", config.SystemEmail),
		sql.Named("twilio_sid", config.TwilioSID),
		sql.Named("twilio_token", config.TwilioToken),
		sql.Named("twilio_from_tel", config.TwilioFromTel),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)

	return errors.Tracef(err)
}
