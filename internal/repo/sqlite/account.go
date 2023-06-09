package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"strings"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
)

type AccountStore struct {
	db *DB
}

func NewAccountStore(ctx context.Context, db *sql.DB) (*AccountStore, error) {
	migrations, err := fs.Sub(migrations, "migrations/account")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := migrateFS(ctx, db, "account", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	s := AccountStore{db: newDB(db)}

	return &s, nil
}

func (s *AccountStore) FindUserByID(ctx context.Context, id int) (*account.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	users, _, err := s.findUsers(ctx, tx, account.UserFilter{ID: &id})
	if err != nil {
		return nil, errors.Tracef(err)
	}
	if len(users) == 0 {
		return nil, errors.Tracef(repo.ErrNotFound)
	}

	user := users[0]
	if err := s.attachUserRecoveryCodes(ctx, tx, user); err != nil {
		return nil, errors.Tracef(err)
	}
	if err := s.attachUserRoles(ctx, tx, user); err != nil {
		return nil, errors.Tracef(err)
	}
	for _, role := range user.Roles {
		if err := s.attachRolePermissions(ctx, tx, role); err != nil {
			return nil, errors.Tracef(err)
		}
	}

	return user, nil
}

func (s *AccountStore) FindUserByEmail(ctx context.Context, email string) (*account.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	users, _, err := s.findUsers(ctx, tx, account.UserFilter{Email: &email})
	if err != nil {
		return nil, errors.Tracef(err)
	}
	if len(users) == 0 {
		return nil, errors.Tracef(repo.ErrNotFound)
	}

	user := users[0]
	if err := s.attachUserRecoveryCodes(ctx, tx, user); err != nil {
		return nil, errors.Tracef(err)
	}
	if err := s.attachUserRoles(ctx, tx, user); err != nil {
		return nil, errors.Tracef(err)
	}

	return user, nil
}

func (s *AccountStore) CountUsersByRoleID(ctx context.Context, roleID int) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Tracef(err)
	}
	defer tx.Rollback()

	_, total, err := s.findUsers(ctx, tx, account.UserFilter{RoleID: &roleID})

	return total, errors.Tracef(err)
}

func (s *AccountStore) FindUsersPageBySearch(ctx context.Context, sortTopID int, search string, page, size int) ([]*account.User, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)
	users, total, err := s.findUsers(ctx, tx, account.UserFilter{
		Search:    &search,
		SortTopID: sortTopID,
		Limit:     limit,
		Offset:    offset,
	})

	return users, total, errors.Tracef(err)
}

func (s *AccountStore) FindRoleByID(ctx context.Context, roleID int) (*account.Role, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	roles, _, err := s.findRoles(ctx, tx, account.RoleFilter{ID: &roleID})
	if len(roles) == 0 {
		return nil, errors.Tracef(repo.ErrNotFound)
	}

	role := roles[0]
	if err := s.attachRolePermissions(ctx, tx, role); err != nil {
		return nil, errors.Tracef(err)
	}

	return role, nil
}

func (s *AccountStore) FindRoleByName(ctx context.Context, name string) (*account.Role, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	roles, _, err := s.findRoles(ctx, tx, account.RoleFilter{Name: &name})
	if len(roles) == 0 {
		return nil, errors.Tracef(repo.ErrNotFound)
	}

	role := roles[0]
	if err := s.attachRolePermissions(ctx, tx, role); err != nil {
		return nil, errors.Tracef(err)
	}

	return role, nil
}

func (s *AccountStore) FindRolesByUserID(ctx context.Context, userID int) ([]*account.Role, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	roles, _, err := s.findRoles(ctx, tx, account.RoleFilter{UserID: &userID})

	return roles, errors.Tracef(err)
}

func (s *AccountStore) FindRoles(ctx context.Context, sortTopID int) ([]*account.Role, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer tx.Rollback()

	users, total, err := s.findRoles(ctx, tx, account.RoleFilter{SortTopID: sortTopID})

	return users, total, errors.Tracef(err)
}

func (s *AccountStore) FindRolesPageBySearch(ctx context.Context, sortTopID int, search string, page, size int) ([]*account.Role, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)
	users, total, err := s.findRoles(ctx, tx, account.RoleFilter{
		Search:    &search,
		SortTopID: sortTopID,
		Limit:     limit,
		Offset:    offset,
	})

	return users, total, errors.Tracef(err)
}

func (s *AccountStore) AddRole(ctx context.Context, role *account.Role) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := s.addRole(ctx, tx, role); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *AccountStore) SaveRole(ctx context.Context, role *account.Role) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := s.saveRole(ctx, tx, role); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *AccountStore) RemoveRole(ctx context.Context, roleID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := s.removeRole(ctx, tx, roleID); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *AccountStore) FindRecoveryCodesByUserID(ctx context.Context, userID int) ([]account.RecoveryCode, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Tracef(err)
	}
	defer tx.Rollback()

	recoveryCodes, _, err := s.findRecoveryCodes(ctx, tx, userID)

	return recoveryCodes, errors.Tracef(err)
}

func (s *AccountStore) AddUser(ctx context.Context, user *account.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := s.addUser(ctx, tx, user); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *AccountStore) SaveUser(ctx context.Context, user *account.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if err := s.saveUser(ctx, tx, user); err != nil {
		return errors.Tracef(err)
	}

	return errors.Tracef(tx.Commit())
}

func (s *AccountStore) findUsers(ctx context.Context, tx *Tx, filter account.UserFilter) ([]*account.User, int, error) {
	var joins []string
	var where []string
	var args []any

	if v := filter.ID; v != nil {
		where, args = append(where, "u.id = ?"), append(args, *v)
	}
	if v := filter.Email; v != nil {
		where, args = append(where, "u.email = ?"), append(args, *v)
	}
	if v := filter.Search; v != nil && *v != "" {
		where, args = append(where, "u.email LIKE ?"), append(args, "%"+*v+"%")
	}
	if v := filter.RoleID; v != nil {
		joins = append(joins, "INNER JOIN account__user_roles AS ur ON u.id = ur.user_id")
		where, args = append(where, "ur.role_id = ?"), append(args, *v)
	}

	var sorts []string
	if filter.SortTopID != 0 {
		sorts, args = append(sorts, "CASE id WHEN ? THEN 0 ELSE 1 END ASC"), append(args, filter.SortTopID)
	}
	sorts = append(sorts, "email ASC")

	rows, err := tx.QueryContext(ctx, `
		SELECT
			u.id,
			u.email,
			u.hashed_password,
			u.totp_method,
			u.totp_tel,
			u.totp_key,
			u.totp_algorithm,
			u.totp_digits,
			u.totp_period_ns,
			u.totp_verified_at,
			u.totp_activated_at,
			u.signed_up_at,
			u.activated_at,
			u.last_signed_in_at,
			u.last_signed_in_method,
			COUNT(1) OVER () AS total
		FROM account__users AS u
		`+strings.Join(joins, "\n")+`
		`+whereSQL(where)+`
		`+orderBySQL(sorts)+`
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
			&user.TOTPTel,
			&user.TOTPKey,
			&user.TOTPAlgorithm,
			&user.TOTPDigits,
			&user.TOTPPeriod,
			(*NullTime)(&user.TOTPVerifiedAt),
			(*NullTime)(&user.TOTPActivatedAt),
			(*Time)(&user.SignedUpAt),
			(*NullTime)(&user.ActivatedAt),
			(*NullTime)(&user.LastSignedInAt),
			&user.LastSignedInMethod,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		users = append(users, &user)
	}

	return users, total, errors.Tracef(rows.Err())
}

func (s *AccountStore) findPermissions(ctx context.Context, tx *Tx, roleID int) ([]string, int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			name,
			COUNT(1) OVER () AS total
		FROM account__permissions AS p
		INNER JOIN account__role_permissions AS rp ON p.id = rp.permission_id
		WHERE rp.role_id = ?`,
		roleID,
	)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer rows.Close()

	var total int
	var permissions []string
	for rows.Next() {
		var permission string

		err := rows.Scan(
			&permission,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		permissions = append(permissions, permission)
	}

	return permissions, total, errors.Tracef(rows.Err())
}

func (s *AccountStore) addPermission(ctx context.Context, tx *Tx, name string) (int, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM account__permissions WHERE name = :name",
		sql.Named("name", name),
	).Scan(&id)
	switch {
	case err == nil:
		return int(id), nil

	case err != nil && !errors.Is(err, repo.ErrNotFound):
		return 0, errors.Tracef(err)
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO account__permissions (
			name,
			created_at
		) VALUES (
			:name,
			:created_at
		)
	`,
		sql.Named("name", name),
		sql.Named("created_at", Time(tx.now.UTC())),
	)
	if err != nil {
		return 0, errors.Tracef(err)
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, errors.Tracef(err)
	}

	return int(id), nil
}

func (s *AccountStore) attachRolePermissions(ctx context.Context, tx *Tx, role *account.Role) error {
	permissions, _, err := s.findPermissions(ctx, tx, role.ID)
	if err != nil {
		return errors.Tracef(err)
	}

	role.Permissions = permissions

	return nil
}

func (s *AccountStore) findRoles(ctx context.Context, tx *Tx, filter account.RoleFilter) ([]*account.Role, int, error) {
	var joins []string
	var where []string
	var args []any

	if v := filter.ID; v != nil {
		where, args = append(where, "r.id = ?"), append(args, *v)
	}
	if v := filter.UserID; v != nil {
		joins = append(joins, "INNER JOIN account__user_roles AS ur ON r.id = ur.role_id")
		where, args = append(where, "ur.user_id = ?"), append(args, *v)
	}
	if v := filter.Name; v != nil && *v != "" {
		where, args = append(where, "r.name = ?"), append(args, *v)
	}
	if v := filter.Search; v != nil && *v != "" {
		where, args = append(where, "r.name LIKE ?"), append(args, "%"+*v+"%")
	}

	var sorts []string
	if filter.SortTopID != 0 {
		sorts, args = append(sorts, "CASE r.id WHEN ? THEN 0 ELSE 1 END ASC"), append(args, filter.SortTopID)
	}
	sorts = append(sorts, "name ASC")

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name,
			description,
			COUNT(1) OVER () AS total
		FROM account__roles AS r
		`+strings.Join(joins, "\n")+`
		`+whereSQL(where)+`
		`+orderBySQL(sorts)+`
		`+limitOffsetSQL(filter.Limit, filter.Offset),
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
			&role.Description,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		roles = append(roles, &role)
	}

	return roles, total, errors.Tracef(rows.Err())
}

func (s *AccountStore) addRole(ctx context.Context, tx *Tx, role *account.Role) error {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO account__roles (
			name,
			description,
			created_at
		) VALUES (
			:name,
			:description,
			:created_at
		)
	`,
		sql.Named("name", role.Name),
		sql.Named("description", role.Description),
		sql.Named("created_at", Time(tx.now.UTC())),
	)
	if err != nil {
		if errors.Is(err, repo.ErrConflict) {
			return errors.Tracef(err, &repo.ConflictError{
				Map: errors.Map{"name": errors.New("already in use")},
			})
		}

		return errors.Tracef(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return errors.Tracef(err)
	}
	role.ID = int(id)

	for _, name := range role.Permissions {
		permissionID, err := s.addPermission(ctx, tx, name)
		if err != nil {
			return errors.Tracef(err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__role_permissions (
				role_id,
				permission_id,
				created_at
			) VALUES (
				:role_id,
				:permission_id,
				:created_at
			)
		`,
			sql.Named("role_id", role.ID),
			sql.Named("permission_id", permissionID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	return nil
}

func (s *AccountStore) saveRole(ctx context.Context, tx *Tx, role *account.Role) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE account__roles SET
			name = :name,
			description = :description,
			updated_at = :updated_at
		WHERE id = :id
	`,
		sql.Named("id", role.ID),
		sql.Named("name", role.Name),
		sql.Named("description", role.Description),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)
	if errors.Is(err, repo.ErrConflict) {
		return errors.Tracef(err, &repo.ConflictError{
			Map: errors.Map{"name": errors.New("already in use")},
		})
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__role_permissions WHERE role_id = :role_id",
		sql.Named("role_id", role.ID),
	)
	if err != nil {
		return errors.Tracef(err)
	}

	for _, name := range role.Permissions {
		permissionID, err := s.addPermission(ctx, tx, name)
		if err != nil {
			return errors.Tracef(err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__role_permissions (
				role_id,
				permission_id,
				created_at
			) VALUES (
				:role_id,
				:permission_id,
				:created_at
			)
		`,
			sql.Named("role_id", role.ID),
			sql.Named("permission_id", permissionID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	return errors.Tracef(err)
}

func (s *AccountStore) removeRole(ctx context.Context, tx *Tx, roleID int) error {
	_, err := tx.ExecContext(ctx,
		"DELETE FROM account__roles WHERE id = :id",
		sql.Named("id", roleID),
	)

	return errors.Tracef(err)
}

func (s *AccountStore) findRecoveryCodes(ctx context.Context, tx *Tx, userID int) ([]account.RecoveryCode, int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			code,
			COUNT(1) OVER () AS total
		FROM account__recovery_codes
		WHERE user_id = :user_id
	`,
		sql.Named("user_id", userID),
	)
	if err != nil {
		return nil, 0, errors.Tracef(err)
	}
	defer rows.Close()

	var total int
	var recoveryCodes []account.RecoveryCode
	for rows.Next() {
		var recoveryCode account.RecoveryCode

		err := rows.Scan(
			&recoveryCode,
			&total,
		)
		if err != nil {
			return nil, 0, errors.Tracef(err)
		}

		recoveryCodes = append(recoveryCodes, recoveryCode)
	}

	return recoveryCodes, total, errors.Tracef(rows.Err())
}

func (s *AccountStore) attachUserRecoveryCodes(ctx context.Context, tx *Tx, user *account.User) error {
	recoveryCodes, _, err := s.findRecoveryCodes(ctx, tx, user.ID)
	if err != nil {
		return errors.Tracef(err)
	}

	if recoveryCodes != nil {
		user.RecoveryCodes = make([]string, len(recoveryCodes))

		for i, rc := range recoveryCodes {
			user.RecoveryCodes[i] = rc.String()
		}
	}

	return nil
}

func (s *AccountStore) attachUserRoles(ctx context.Context, tx *Tx, user *account.User) error {
	roles, _, err := s.findRoles(ctx, tx, account.RoleFilter{UserID: &user.ID})
	if err != nil {
		return errors.Tracef(err)
	}

	user.Roles = roles

	return nil
}

func (s *AccountStore) addUser(ctx context.Context, tx *Tx, user *account.User) error {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO account__users (
			email,
			hashed_password,
			totp_method,
			totp_tel,
			totp_key,
			totp_algorithm,
			totp_digits,
			totp_period_ns,
			totp_verified_at,
			totp_activated_at,
			signed_up_at,
			activated_at,
			last_signed_in_at,
			last_signed_in_method,
			created_at
		) VALUES (
			:email,
			:hashed_password,
			:totp_method,
			:totp_tel,
			:totp_key,
			:totp_algorithm,
			:totp_digits,
			:totp_period_ns,
			:totp_verified_at,
			:totp_activated_at,
			:signed_up_at,
			:activated_at,
			:last_signed_in_at,
			:last_signed_in_method,
			:created_at
		)
	`,
		sql.Named("email", user.Email),
		sql.Named("hashed_password", user.HashedPassword),
		sql.Named("totp_method", user.TOTPMethod),
		sql.Named("totp_tel", user.TOTPTel),
		sql.Named("totp_key", user.TOTPKey),
		sql.Named("totp_algorithm", user.TOTPAlgorithm),
		sql.Named("totp_digits", user.TOTPDigits),
		sql.Named("totp_period_ns", user.TOTPPeriod),
		sql.Named("totp_verified_at", NullTime(user.TOTPVerifiedAt)),
		sql.Named("totp_activated_at", NullTime(user.TOTPActivatedAt)),
		sql.Named("signed_up_at", Time(user.SignedUpAt)),
		sql.Named("activated_at", NullTime(user.ActivatedAt)),
		sql.Named("last_signed_in_at", NullTime(user.LastSignedInAt)),
		sql.Named("last_signed_in_method", user.LastSignedInMethod),
		sql.Named("created_at", Time(tx.now.UTC())),
	)
	if err != nil {
		if errors.Is(err, repo.ErrConflict) {
			return errors.Tracef(err, &repo.ConflictError{
				Map: errors.Map{"email": errors.New("already in use")},
			})
		}

		return errors.Tracef(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return errors.Tracef(err)
	}
	user.ID = int(id)

	for _, rc := range user.RecoveryCodes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__recovery_codes (
				user_id,
				code,
				created_at
			) VALUES (
				:user_id,
				:code,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("code", rc),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	for _, role := range user.Roles {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__user_roles (
				user_id,
				role_id,
				created_at
			) VALUES (
				:user_id,
				:role_id,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("role_id", role.ID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	return nil
}

func (s *AccountStore) saveUser(ctx context.Context, tx *Tx, user *account.User) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE account__users SET
			email = :email,
			hashed_password = :hashed_password,
			totp_method = :totp_method,
			totp_tel = :totp_tel,
			totp_key = :totp_key,
			totp_algorithm = :totp_algorithm,
			totp_digits = :totp_digits,
			totp_period_ns = :totp_period_ns,
			totp_verified_at = :totp_verified_at,
			totp_activated_at = :totp_activated_at,
			signed_up_at = :signed_up_at,
			activated_at = :activated_at,
			last_signed_in_at = :last_signed_in_at,
			last_signed_in_method = :last_signed_in_method,
			updated_at = :updated_at
		WHERE id = :id
	`,
		sql.Named("id", user.ID),
		sql.Named("email", user.Email),
		sql.Named("hashed_password", user.HashedPassword),
		sql.Named("totp_method", user.TOTPMethod),
		sql.Named("totp_tel", user.TOTPTel),
		sql.Named("totp_key", user.TOTPKey),
		sql.Named("totp_algorithm", user.TOTPAlgorithm),
		sql.Named("totp_digits", user.TOTPDigits),
		sql.Named("totp_period_ns", user.TOTPPeriod),
		sql.Named("totp_verified_at", NullTime(user.TOTPVerifiedAt)),
		sql.Named("totp_activated_at", NullTime(user.TOTPActivatedAt)),
		sql.Named("signed_up_at", Time(user.SignedUpAt)),
		sql.Named("activated_at", NullTime(user.ActivatedAt)),
		sql.Named("last_signed_in_at", NullTime(user.LastSignedInAt)),
		sql.Named("last_signed_in_method", user.LastSignedInMethod),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)
	if err != nil {
		if errors.Is(err, repo.ErrConflict) {
			return errors.Tracef(err, &repo.ConflictError{
				Map: errors.Map{"email": errors.New("already in use")},
			})
		}

		return errors.Tracef(err)
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__recovery_codes WHERE user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return errors.Tracef(err)
	}

	for _, rc := range user.RecoveryCodes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__recovery_codes (
				user_id,
				code,
				created_at
			) VALUES (
				:user_id,
				:code,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("code", rc),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__user_roles WHERE user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return errors.Tracef(err)
	}

	for _, role := range user.Roles {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__user_roles (
				user_id,
				role_id,
				created_at
			) VALUES (
				:user_id,
				:role_id,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("role_id", role.ID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return errors.Tracef(err)
		}
	}

	return nil
}
