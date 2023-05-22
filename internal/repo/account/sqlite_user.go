package account

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/aesgcm"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account"
)

type sqliteUser struct {
	ID             string        `sql:"id"`
	Email          string        `sql:"email"`
	HashedPassword []byte        `sql:"hashed_password"`
	TOTPUseSMS     bool          `sql:"totp_use_sms"`
	TOTPTelephone  string        `sql:"totp_telephone"`
	TOTPKey        []byte        `sql:"totp_key"`
	TOTPAlgorithm  string        `sql:"totp_algorithm"`
	TOTPDigits     int           `sql:"totp_digits"`
	TOTPPeriod     time.Duration `sql:"totp_period"`
	TOTPVerifiedAt sql.NullTime  `sql:"totp_verified_at"`
	ActivatedAt    sql.NullTime  `sql:"activated_at"`
}

type UserRepo struct {
	db     *sqlite.DB
	secret []byte
}

func NewSQLiteUserRepo(ctx context.Context, db *sqlite.DB, secret []byte) (*UserRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/sqlite")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := db.MigrateFS(ctx, "account", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	repo := UserRepo{
		db:     db,
		secret: secret,
	}

	return &repo, nil
}

func (r *UserRepo) findBy(ctx context.Context, where string, args sqlite.Args) (account.User, error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return account.User{}, errors.Tracef(err)
	}
	defer tx.Rollback()

	var row sqliteUser
	cols := strings.Join(sqlite.Columns(&row), ", ")
	stmt := fmt.Sprintf("SELECT %v FROM account__users WHERE %v;", cols, where)
	if err := r.db.QueryRow(ctx, stmt, args).ScanInto(&row); err != nil {
		return account.User{}, errors.Tracef(err)
	}

	if row.TOTPKey != nil {
		decryptedTOTPKey, err := aesgcm.Decrypt(r.secret, row.TOTPKey)
		if err != nil {
			return account.User{}, errors.Tracef(err)
		}

		row.TOTPKey = decryptedTOTPKey
	}

	recoveryCodes, err := r.findRecoveryCodesByID(ctx, tx, row.ID)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return account.User{}, errors.Tracef(err)
	}

	roles, err := r.findRolesByID(ctx, tx, row.ID)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return account.User{}, errors.Tracef(err)
	}

	id, err := uuid.ParseV4(row.ID)
	if err != nil {
		return account.User{}, errors.Tracef(err)
	}

	res := account.NewUser(id)

	res.Email = text.Email(row.Email)
	res.HashedPassword = row.HashedPassword
	res.TOTPUseSMS = row.TOTPUseSMS
	res.TOTPTelephone = text.Telephone(row.TOTPTelephone)
	res.TOTPKey = row.TOTPKey
	res.TOTPAlgorithm = row.TOTPAlgorithm
	res.TOTPDigits = row.TOTPDigits
	res.TOTPPeriod = row.TOTPPeriod
	res.TOTPVerifiedAt = row.TOTPVerifiedAt.Time
	res.RecoveryCodes = recoveryCodes
	res.Roles = roles
	res.ActivatedAt = row.ActivatedAt.Time

	return res, errors.Tracef(tx.Commit())
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.V4) (account.User, error) {
	return r.findBy(ctx, `id = :id`, sqlite.Args{"id": id})
}

func (r *UserRepo) FindByEmail(ctx context.Context, email text.Email) (account.User, error) {
	return r.findBy(ctx, `email = :email`, sqlite.Args{"email": email})
}

func (r *UserRepo) Add(ctx context.Context, u account.User) error {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	encryptedTOTPKey, err := aesgcm.Encrypt(r.secret, u.TOTPKey)
	if err != nil {
		return errors.Tracef(err)
	}

	stmt, args := `
		INSERT INTO account__users
			(id, email, hashed_password, totp_use_sms, totp_telephone, totp_key, totp_algorithm, totp_digits, totp_period)
		VALUES
			(:id, :email, :hashed_password, :totp_use_sms, :totp_telephone, :totp_key, :totp_algorithm, :totp_digits, :totp_period);
	`, sqlite.Args{
		"id":              u.ID,
		"email":           u.Email,
		"hashed_password": u.HashedPassword,
		"totp_use_sms":    u.TOTPUseSMS,
		"totp_telephone":  u.TOTPTelephone,
		"totp_key":        encryptedTOTPKey,
		"totp_algorithm":  u.TOTPAlgorithm,
		"totp_digits":     u.TOTPDigits,
		"totp_period":     u.TOTPPeriod,
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return errors.Tracef(err)
	}

	for _, code := range u.RecoveryCodes {
		encrypted, err := aesgcm.Encrypt(r.secret, []byte(code))
		if err != nil {
			return errors.Tracef(err)
		}

		stmt, args := `
			INSERT INTO account__recovery_codes
				(user_id, code)
			VALUES
				(:user_id, :code);
		`, sqlite.Args{
			"user_id": u.ID,
			"code":    encrypted,
		}
		if _, err := tx.Exec(ctx, stmt, args); err != nil {
			return errors.Tracef(err)
		}
	}

	// for _, role := range u.Roles {
	// 	var roleID uuid.V4

	// 	stmt := "SELECT id FROM roles WHERE name = ?"
	// 	err := tx.QueryRow(ctx, stmt, role.Name).Scan(&roleID)
	// 	if err != nil {
	// 		return errors.Tracef(err)
	// 	}

	// 	stmt = "INSERT INTO user_roles (user_id, role_id) VALUES (?, ?);"
	// 	_, err = tx.Exec(ctx, stmt, u.ID, roleID)
	// 	if err != nil {
	// 		return errors.Tracef(err)
	// 	}
	// }

	return errors.Tracef(tx.Commit())
}

func (r *UserRepo) Save(ctx context.Context, u account.User) error {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	encryptedTOTPKey, err := aesgcm.Encrypt(r.secret, u.TOTPKey)
	if err != nil {
		return errors.Tracef(err)
	}

	stmt, args := `
		UPDATE account__users SET
			email = :email,
			hashed_password = :hashed_password,
			totp_use_sms = :totp_use_sms,
			totp_telephone = :totp_telephone,
			totp_key = :totp_key,
			totp_algorithm = :totp_algorithm,
			totp_digits = :totp_digits,
			totp_period = :totp_period,
			totp_verified_at = :totp_verified_at,
			activated_at = :activated_at
		WHERE id = :id;
	`, sqlite.Args{
		"id":               u.ID,
		"email":            u.Email,
		"hashed_password":  u.HashedPassword,
		"totp_use_sms":     u.TOTPUseSMS,
		"totp_telephone":   u.TOTPTelephone,
		"totp_key":         encryptedTOTPKey,
		"totp_algorithm":   u.TOTPAlgorithm,
		"totp_digits":      u.TOTPDigits,
		"totp_period":      u.TOTPPeriod,
		"totp_verified_at": sqlite.NewNullTime(u.TOTPVerifiedAt.UTC()),
		"activated_at":     sqlite.NewNullTime(u.ActivatedAt.UTC()),
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return errors.Tracef(err)
	}

	stmt, args = `
		DELETE FROM account__recovery_codes
		WHERE user_id = :user_id;
	`, sqlite.Args{
		"user_id": u.ID,
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return errors.Tracef(err)
	}

	for _, code := range u.RecoveryCodes {
		encrypted, err := aesgcm.Encrypt(r.secret, []byte(code))
		if err != nil {
			return errors.Tracef(err)
		}

		stmt, args := `
			INSERT INTO account__recovery_codes
				(user_id, code)
			VALUES
				(:user_id, :code);
		`, sqlite.Args{
			"user_id": u.ID,
			"code":    encrypted,
		}
		if _, err := tx.Exec(ctx, stmt, args); err != nil {
			return errors.Tracef(err)
		}
	}

	return errors.Tracef(tx.Commit())
}

func (r *UserRepo) findRolesByID(ctx context.Context, db sqlite.Querier, userID string) ([]account.Role, error) {
	var roles []account.Role

	stmt, args := `
		SELECT id, name
		FROM account__roles AS r
		INNER JOIN account__user_roles AS ur ON r.id = ur.role_id
		WHERE ur.user_id = :user_id;
	`, sqlite.Args{"user_id": userID}
	rows, err := db.Query(ctx, stmt, args)
	if err != nil {
		return roles, errors.Tracef(err)
	}
	for rows.Next() {
		var roleID uuid.V4
		var roleName string

		if err := rows.Scan(&roleID, &roleName); err != nil {
			return roles, errors.Tracef(err)
		}

		var permissions []account.Permission

		stmt, args := `
			SELECT id
			FROM account__permissions AS p
			INNER JOIN account__role_permissions AS rp ON p.id = rp.permission_id
			WHERE rp.role_id = :role_id;
		`, sqlite.Args{"role_id": roleID}
		rows, err := db.Query(ctx, stmt, args)
		if err != nil {
			return roles, errors.Tracef(err)
		}
		for rows.Next() {
			var permissionID account.Permission

			if err := rows.Scan(&permissionID); err != nil {
				return roles, errors.Tracef(err)
			}

			permissions = append(permissions, permissionID)
		}
		if err := rows.Err(); err != nil {
			return roles, errors.Tracef(err)
		}

		roles = append(roles, account.NewRole(roleName, permissions...))
	}

	return roles, errors.Tracef(rows.Err())
}

func (r *UserRepo) findRecoveryCodesByID(ctx context.Context, db sqlite.Querier, userID string) ([]account.RecoveryCode, error) {
	var recoveryCodes []account.RecoveryCode

	stmt, args := `
		SELECT code
		FROM account__recovery_codes
		WHERE user_id = :user_id;
	`, sqlite.Args{"user_id": userID}
	rows, err := db.Query(ctx, stmt, args)
	if err != nil {
		return recoveryCodes, errors.Tracef(err)
	}
	for rows.Next() {
		var encrypted []byte

		if err := rows.Scan(&encrypted); err != nil {
			return recoveryCodes, errors.Tracef(err)
		}

		decrypted, err := aesgcm.Decrypt(r.secret, encrypted)
		if err != nil {
			return recoveryCodes, errors.Tracef(err)
		}

		recoveryCode, err := account.NewRecoveryCode(string(decrypted))
		if err != nil {
			return recoveryCodes, errors.Tracef(err)
		}

		recoveryCodes = append(recoveryCodes, recoveryCode)
	}

	return recoveryCodes, errors.Tracef(rows.Err())
}
