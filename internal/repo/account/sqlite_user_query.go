package account

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/polyscone/tofu/internal/adapter/web/query"
	"github.com/polyscone/tofu/internal/pkg/aesgcm"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

type UserQueryRepo struct {
	db     *sqlite.DB
	secret []byte
}

func NewSQLiteUserQueryRepo(ctx context.Context, db *sqlite.DB, secret []byte) (*UserQueryRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/sqlite")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := db.MigrateFS(ctx, "account", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	repo := UserQueryRepo{
		db:     db,
		secret: secret,
	}

	return &repo, nil
}

func (r *UserQueryRepo) findBy(ctx context.Context, where string, args sqlite.Args) (query.AccountUser, error) {
	var res query.AccountUser
	var activatedAt sql.NullTime

	stmt := fmt.Sprintf(`
		SELECT u.id, u.email, u.totp_use_sms, u.totp_telephone, u.activated_at
		FROM account__users AS u
		WHERE %v;
	`, where)
	err := r.db.QueryRow(ctx, stmt, args).Scan(
		&res.ID,
		&res.Email,
		&res.TOTPUseSMS,
		&res.TOTPTelephone,
		&activatedAt,
	)
	if err != nil {
		return res, errors.Tracef(err)
	}

	res.ActivatedAt = activatedAt.Time

	return res, nil
}

func (r *UserQueryRepo) FindByID(ctx context.Context, userID string) (query.AccountUser, error) {
	return r.findBy(ctx, "u.id = :user_id", sqlite.Args{"user_id": userID})
}

func (r *UserQueryRepo) FindByEmail(ctx context.Context, email string) (query.AccountUser, error) {
	return r.findBy(ctx, "u.email = :email", sqlite.Args{"email": email})
}

func (r *UserQueryRepo) FindByPageFilter(ctx context.Context, page, size int, filter string) (*repo.Book[query.AccountUser], error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	var where string
	var args sqlite.Args
	if filter != "" {
		where = "WHERE email LIKE :filter"
		args = sqlite.Args{"filter": "%" + filter + "%"}
	}

	var count int
	stmt := fmt.Sprintf("SELECT COUNT(1) FROM account__users %v;", where)
	if err := tx.QueryRow(ctx, stmt, args).Scan(&count); err != nil {
		return nil, errors.Tracef(err)
	}

	res := repo.NewBook[query.AccountUser](page, size, count)

	stmt = fmt.Sprintf(`
		SELECT id, email, totp_use_sms, totp_telephone, activated_at
		FROM account__users
		%v
		LIMIT %v, %v;
	`, where, (page-1)*size, size)
	rows, err := r.db.Query(ctx, stmt, args)
	if err != nil {
		return res, errors.Tracef(err)
	}

	for rows.Next() {
		var row query.AccountUser
		var activatedAt sql.NullTime

		err := rows.Scan(
			&row.ID,
			&row.Email,
			&row.TOTPUseSMS,
			&row.TOTPTelephone,
			&activatedAt,
		)
		if err != nil {
			return res, errors.Tracef(err)
		}

		row.ActivatedAt = activatedAt.Time

		res.AddRow(row)
	}

	return res, errors.Tracef(tx.Commit())
}

func (r *UserQueryRepo) FindTOTPParamsByID(ctx context.Context, userID string) (query.TOTPParams, error) {
	var res query.TOTPParams

	stmt, args := `
		SELECT u.totp_key, u.totp_algorithm, u.totp_digits, u.totp_period
		FROM account__users AS u
		WHERE u.id = :user_id;
	`, sqlite.Args{
		"user_id": userID,
	}
	err := r.db.QueryRow(ctx, stmt, args).Scan(&res.Key, &res.Algorithm, &res.Digits, &res.Period)
	if err != nil {
		return res, errors.Tracef(err)
	}

	decryptedTOTPKey, err := aesgcm.Decrypt(r.secret, res.Key)
	if err != nil {
		return res, errors.Tracef(err)
	}
	res.Key = decryptedTOTPKey

	return res, nil
}

func (r *UserQueryRepo) FindRecoveryCodesByID(ctx context.Context, userID string) ([]string, error) {
	var res []string

	stmt, args := `
		SELECT code
		FROM account__recovery_codes
		WHERE user_id = :user_id;
	`, sqlite.Args{
		"user_id": userID,
	}
	rows, err := r.db.Query(ctx, stmt, args)
	if err != nil {
		return res, errors.Tracef(err)
	}

	for rows.Next() {
		var encrypted []byte

		if err := rows.Scan(&encrypted); err != nil {
			return res, errors.Tracef(err)
		}

		decrypted, err := aesgcm.Decrypt(r.secret, encrypted)
		if err != nil {
			return res, errors.Tracef(err)
		}

		res = append(res, string(decrypted))
	}

	return res, nil
}
