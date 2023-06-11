package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/repo"
)

type SystemStore struct {
	db *DB
}

func NewSystemStore(ctx context.Context, db *sql.DB) (*SystemStore, error) {
	migrations, err := fs.Sub(migrations, "migrations/system")
	if err != nil {
		return nil, fmt.Errorf("initialise system migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db, "system", migrations); err != nil {
		return nil, fmt.Errorf("migrate system: %w", err)
	}

	s := SystemStore{db: newDB(db)}

	return &s, nil
}

func (s *SystemStore) FindConfig(ctx context.Context) (*system.Config, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return s.findConfig(ctx, tx)
}

func (s *SystemStore) SaveConfig(ctx context.Context, config *system.Config) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.upsertConfig(ctx, tx, config); err != nil {
		return fmt.Errorf("upsert config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
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
		return nil, err
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

	return err
}
