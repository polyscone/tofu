package account

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/polyscone/tofu/internal/adapter/web/query"
	"github.com/polyscone/tofu/internal/pkg/aesgcm"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

func (u *sqliteUser) newWebAccountUser() query.AccountUser {
	return query.AccountUser{
		ID:            u.ID,
		Email:         u.Email,
		TOTPUseSMS:    u.TOTPUseSMS,
		TOTPTelephone: u.TOTPTelephone,
		ActivatedAt:   u.ActivatedAt.Time,
	}
}

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

func (r *UserQueryRepo) findBy(ctx context.Context, cols string, where string, args sqlite.Args) (sqliteUser, error) {
	var row sqliteUser
	stmt := fmt.Sprintf("SELECT %v FROM account__users WHERE %v;", cols, where)
	if err := r.db.QueryRow(ctx, stmt, args).ScanInto(&row); err != nil {
		return row, errors.Tracef(err)
	}

	if row.TOTPKey != nil {
		decryptedTOTPKey, err := aesgcm.Decrypt(r.secret, row.TOTPKey)
		if err != nil {
			return row, errors.Tracef(err)
		}

		row.TOTPKey = decryptedTOTPKey
	}

	return row, nil
}

func (r *UserQueryRepo) FindByID(ctx context.Context, userID string) (query.AccountUser, error) {
	cols := "id, email, totp_use_sms, totp_telephone, activated_at"
	where := "id = :user_id"
	args := sqlite.Args{"user_id": userID}
	row, err := r.findBy(ctx, cols, where, args)

	return row.newWebAccountUser(), errors.Tracef(err)
}

func (r *UserQueryRepo) FindByEmail(ctx context.Context, email string) (query.AccountUser, error) {
	cols := "id, email, totp_use_sms, totp_telephone, activated_at"
	where := "email = :email"
	args := sqlite.Args{"email": email}
	row, err := r.findBy(ctx, cols, where, args)

	return row.newWebAccountUser(), errors.Tracef(err)
}

func (r *UserQueryRepo) FindByPage(ctx context.Context, page, size int, filter string) (*repo.Book[query.AccountUser], error) {
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

	stmt = fmt.Sprintf(
		"SELECT id, email, totp_use_sms, totp_telephone, activated_at FROM account__users %v LIMIT %v, %v;",
		where, (page-1)*size, size,
	)
	rows, err := r.db.Query(ctx, stmt, args)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	book := repo.NewBook[query.AccountUser](page, size, count)
	for rows.Next() {
		var row sqliteUser
		if err := rows.ScanInto(&row); err != nil {
			return book, errors.Tracef(err)
		}

		book.AddRow(row.newWebAccountUser())
	}

	return book, errors.Tracef(tx.Commit())
}

func (r *UserQueryRepo) FindTOTPParamsByID(ctx context.Context, userID string) (query.TOTPParams, error) {
	var res query.TOTPParams

	cols := "totp_key, totp_algorithm, totp_digits, totp_period"
	where := "id = :user_id"
	args := sqlite.Args{"user_id": userID}
	row, err := r.findBy(ctx, cols, where, args)
	if err != nil {
		return res, errors.Tracef(err)
	}

	res = query.TOTPParams{
		Key:       row.TOTPKey,
		Algorithm: row.TOTPAlgorithm,
		Digits:    row.TOTPDigits,
		Period:    row.TOTPPeriod,
	}

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
		return nil, errors.Tracef(err)
	}

	for rows.Next() {
		var encrypted []byte
		if err := rows.Scan(&encrypted); err != nil {
			return nil, errors.Tracef(err)
		}

		decrypted, err := aesgcm.Decrypt(r.secret, encrypted)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		res = append(res, string(decrypted))
	}

	return res, nil
}
