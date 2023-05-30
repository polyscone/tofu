package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/aesgcm"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
)

type AccountRepo struct {
	db     *DB
	secret []byte
}

func NewAccountRepo(ctx context.Context, db *sql.DB, secret []byte) (*AccountRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/account")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := migrateFS(ctx, db, "account", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	r := AccountRepo{
		db:     newDB(db),
		secret: secret,
	}

	return &r, nil
}

func (r *AccountRepo) FindUserByID(ctx context.Context, id int) (*account.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	users, _, err := r.findUsers(ctx, tx, account.UserFilter{ID: &id})
	if err != nil {
		return nil, errors.Tracef(err)
	}
	if len(users) == 0 {
		return nil, errors.Tracef(repo.ErrNotFound)
	}

	user := users[0]
	if err := r.attachUserRecoveryCodes(ctx, tx, user); err != nil {
		return nil, errors.Tracef(err)
	}

	return user, nil
}

func (r *AccountRepo) FindUserByEmail(ctx context.Context, email string) (*account.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	users, _, err := r.findUsers(ctx, tx, account.UserFilter{Email: &email})
	if err != nil {
		return nil, errors.Tracef(err)
	}
	if len(users) == 0 {
		return nil, errors.Tracef(repo.ErrNotFound)
	}

	user := users[0]
	if err := r.attachUserRecoveryCodes(ctx, tx, user); err != nil {
		return nil, errors.Tracef(err)
	}

	return user, nil
}

func (r *AccountRepo) FindUsersByPage(ctx context.Context, search string, page, size int) ([]*account.User, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)
	users, total, err := r.findUsers(ctx, tx, account.UserFilter{
		Search: &search,
		Limit:  limit,
		Offset: offset,
	})

	return users, total, errors.Tracef(err)
}

func (r *AccountRepo) FindRolesByUserID(ctx context.Context, userID int) ([]*account.Role, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	roles, _, err := r.findRoles(ctx, tx, account.RoleFilter{UserID: &userID})

	return roles, errors.Tracef(err)
}

func (r *AccountRepo) FindRecoveryCodesByUserID(ctx context.Context, userID int) ([]*account.RecoveryCode, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	recoveryCodes, _, err := r.findRecoveryCodes(ctx, tx, account.RecoveryCodeFilter{UserID: &userID})

	return recoveryCodes, errors.Tracef(err)
}

func (r *AccountRepo) AddUser(ctx context.Context, user *account.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := r.addUser(ctx, tx, user); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *AccountRepo) SaveUser(ctx context.Context, user *account.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := r.saveUser(ctx, tx, user); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (r *AccountRepo) findUsers(ctx context.Context, tx *Tx, filter account.UserFilter) ([]*account.User, int, error) {
	var where []string
	var args []any

	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.Email; v != nil {
		where, args = append(where, "email = ?"), append(args, *v)
	}
	if v := filter.Search; v != nil && *v != "" {
		where, args = append(where, "email LIKE ?"), append(args, "%"+*v+"%")
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			email,
			hashed_password,
			totp_method,
			totp_telephone,
			totp_key,
			totp_algorithm,
			totp_digits,
			totp_period_ns,
			totp_verified_at,
			totp_activated_at,
			registered_at,
			activated_at,
			last_logged_in_at,
			COUNT(1) OVER () AS total
		FROM account__users
		`+whereSQL(where)+`
		`+limitOffsetSQL(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer rows.Close()

	var total int
	var users []*account.User
	for rows.Next() {
		var user account.User

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.HashedPassword,
			&user.TOTPMethod,
			&user.TOTPTelephone,
			&user.TOTPKey,
			&user.TOTPAlgorithm,
			&user.TOTPDigits,
			&user.TOTPPeriod,
			(*NullTime)(&user.TOTPVerifiedAt),
			(*NullTime)(&user.TOTPActivatedAt),
			(*Time)(&user.RegisteredAt),
			(*NullTime)(&user.ActivatedAt),
			(*NullTime)(&user.LastLoggedInAt),
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		if len(user.TOTPKey) != 0 {
			decryptedKey, err := aesgcm.Decrypt(r.secret, user.TOTPKey)
			if err != nil {
				return nil, 0, errors.Tracef(err)
			}

			user.TOTPKey = decryptedKey
		}

		users = append(users, &user)
	}

	return users, total, errors.Tracef(rows.Err())
}

func (r *AccountRepo) findPermissions(ctx context.Context, tx *Tx, filter account.PermissionFilter) ([]*account.Permission, int, error) {
	var where []string
	var args []any

	if v := filter.RoleID; v != nil {
		where, args = append(where, "rp.role_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name,
			COUNT(1) OVER () AS total
		FROM account__permissions AS p
		INNER JOIN account__role_permissions AS rp ON p.id = rp.permission_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer rows.Close()

	var total int
	var permissions []*account.Permission
	for rows.Next() {
		var permission account.Permission

		err := rows.Scan(
			&permission.ID,
			&permission.Name,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		permissions = append(permissions, &permission)
	}

	return permissions, total, errors.Tracef(rows.Err())
}

func (r *AccountRepo) findRoles(ctx context.Context, tx *Tx, filter account.RoleFilter) ([]*account.Role, int, error) {
	var where []string
	var args []any

	if v := filter.UserID; v != nil {
		where, args = append(where, "ur.user_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name,
			COUNT(1) OVER () AS total
		FROM account__roles AS r
		INNER JOIN account__user_roles AS ur ON r.id = ur.role_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer rows.Close()

	var total int
	var roles []*account.Role
	for rows.Next() {
		var role account.Role

		err := rows.Scan(
			&role.ID,
			&role.Name,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		permissions, _, err := r.findPermissions(ctx, tx, account.PermissionFilter{RoleID: &role.ID})
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		role.Permissions = permissions

		roles = append(roles, &role)
	}

	return roles, total, errors.Tracef(rows.Err())
}

func (r *AccountRepo) findRecoveryCodes(ctx context.Context, tx *Tx, filter account.RecoveryCodeFilter) ([]*account.RecoveryCode, int, error) {
	var where []string
	var args []any

	if v := filter.UserID; v != nil {
		where, args = append(where, "user_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			code,
			COUNT(1) OVER () AS total
		FROM account__recovery_codes
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer rows.Close()

	var total int
	var recoveryCodes []*account.RecoveryCode
	for rows.Next() {
		var recoveryCode account.RecoveryCode

		err := rows.Scan(
			&recoveryCode.Code,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		decrypted, err := aesgcm.Decrypt(r.secret, []byte(recoveryCode.Code))
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		recoveryCode.Code = string(decrypted)

		recoveryCodes = append(recoveryCodes, &recoveryCode)
	}

	return recoveryCodes, total, errors.Tracef(rows.Err())
}

func (r *AccountRepo) attachUserRecoveryCodes(ctx context.Context, tx *Tx, user *account.User) error {
	recoveryCodes, _, err := r.findRecoveryCodes(ctx, tx, account.RecoveryCodeFilter{UserID: &user.ID})
	if err != nil {
		return errors.Tracef(err)
	}

	user.RecoveryCodes = recoveryCodes

	return nil
}

func (r *AccountRepo) addUser(ctx context.Context, tx *Tx, u *account.User) error {
	var encryptedTOTPKey []byte
	if u.TOTPKey != nil {
		var err error
		encryptedTOTPKey, err = aesgcm.Encrypt(r.secret, u.TOTPKey)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO account__users (
			email,
			hashed_password,
			totp_method,
			totp_telephone,
			totp_key,
			totp_algorithm,
			totp_digits,
			totp_period_ns,
			totp_verified_at,
			totp_activated_at,
			registered_at,
			activated_at,
			last_logged_in_at
		) VALUES (
			:email,
			:hashed_password,
			:totp_method,
			:totp_telephone,
			:totp_key,
			:totp_algorithm,
			:totp_digits,
			:totp_period_ns,
			:totp_verified_at,
			:totp_activated_at,
			:registered_at,
			:activated_at,
			:last_logged_in_at
		)
	`,
		sql.Named("email", u.Email),
		sql.Named("hashed_password", u.HashedPassword),
		sql.Named("totp_method", u.TOTPMethod),
		sql.Named("totp_telephone", u.TOTPTelephone),
		sql.Named("totp_key", encryptedTOTPKey),
		sql.Named("totp_algorithm", u.TOTPAlgorithm),
		sql.Named("totp_digits", u.TOTPDigits),
		sql.Named("totp_period_ns", u.TOTPPeriod),
		sql.Named("totp_verified_at", NullTime(u.TOTPVerifiedAt.UTC())),
		sql.Named("totp_activated_at", NullTime(u.TOTPActivatedAt.UTC())),
		sql.Named("registered_at", Time(u.RegisteredAt.UTC())),
		sql.Named("activated_at", NullTime(u.ActivatedAt.UTC())),
		sql.Named("last_logged_in_at", NullTime(u.LastLoggedInAt.UTC())),
	)
	if err != nil {
		return errors.Tracef(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return errors.Tracef(err)
	}
	u.ID = int(id)

	for _, rc := range u.RecoveryCodes {
		encrypted, err := aesgcm.Encrypt(r.secret, []byte(rc.Code))
		if err != nil {
			return errors.Tracef(err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__recovery_codes (
				user_id,
				code
			) VALUES (
				:user_id,
				:code
			)
		`,
			sql.Named("user_id", u.ID),
			sql.Named("code", encrypted),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	return nil
}

func (r *AccountRepo) saveUser(ctx context.Context, tx *Tx, u *account.User) error {
	var encryptedTOTPKey []byte
	if u.TOTPKey != nil {
		var err error
		encryptedTOTPKey, err = aesgcm.Encrypt(r.secret, u.TOTPKey)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE account__users SET
			email = :email,
			hashed_password = :hashed_password,
			totp_method = :totp_method,
			totp_telephone = :totp_telephone,
			totp_key = :totp_key,
			totp_algorithm = :totp_algorithm,
			totp_digits = :totp_digits,
			totp_period_ns = :totp_period_ns,
			totp_verified_at = :totp_verified_at,
			totp_activated_at = :totp_activated_at,
			registered_at = :registered_at,
			activated_at = :activated_at,
			last_logged_in_at = :last_logged_in_at,
			updated_at = :updated_at
		WHERE id = :id
	`,
		sql.Named("id", u.ID),
		sql.Named("email", u.Email),
		sql.Named("hashed_password", u.HashedPassword),
		sql.Named("totp_method", u.TOTPMethod),
		sql.Named("totp_telephone", u.TOTPTelephone),
		sql.Named("totp_key", encryptedTOTPKey),
		sql.Named("totp_algorithm", u.TOTPAlgorithm),
		sql.Named("totp_digits", u.TOTPDigits),
		sql.Named("totp_period_ns", u.TOTPPeriod),
		sql.Named("totp_verified_at", NullTime(u.TOTPVerifiedAt.UTC())),
		sql.Named("totp_activated_at", NullTime(u.TOTPActivatedAt.UTC())),
		sql.Named("registered_at", Time(u.RegisteredAt.UTC())),
		sql.Named("activated_at", NullTime(u.ActivatedAt.UTC())),
		sql.Named("last_logged_in_at", NullTime(u.LastLoggedInAt.UTC())),
		sql.Named("updated_at", Time(time.Now().UTC())),
	)
	if err != nil {
		return errors.Tracef(err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM account__recovery_codes
		WHERE user_id = :user_id
	`,
		sql.Named("user_id", u.ID),
	)
	if err != nil {
		return errors.Tracef(err)
	}

	for _, rc := range u.RecoveryCodes {
		encrypted, err := aesgcm.Encrypt(r.secret, []byte(rc.Code))
		if err != nil {
			return errors.Tracef(err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__recovery_codes (
				user_id,
				code
			) VALUES (
				:user_id,
				:code
			)
		`,
			sql.Named("user_id", u.ID),
			sql.Named("code", encrypted),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	return nil
}
