package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository"
)

type AccountRepo struct {
	db *DB
}

func NewAccountRepo(ctx context.Context, db *sql.DB) (*AccountRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/account")
	if err != nil {
		return nil, fmt.Errorf("initialise account migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db, "account", migrations); err != nil {
		return nil, fmt.Errorf("migrate account: %w", err)
	}

	r := AccountRepo{db: newDB(db)}

	return &r, nil
}

func (r *AccountRepo) FindUserByID(ctx context.Context, id int) (*account.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	users, _, err := r.findUsers(ctx, tx, account.UserFilter{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, repository.ErrNotFound
	}

	user := users[0]

	if err := r.attachUserRecoveryCodes(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user recovery codes: %w", err)
	}

	if err := r.attachUserRoles(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user roles: %w", err)
	}
	for _, role := range user.Roles {
		if err := r.attachRolePermissions(ctx, tx, role); err != nil {
			return nil, fmt.Errorf("attach role permissions: %w", err)
		}
	}

	if err := r.attachUserGrants(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user grants: %w", err)
	}

	if err := r.attachUserDenials(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user denials: %w", err)
	}

	return user, nil
}

func (r *AccountRepo) FindUserByEmail(ctx context.Context, email string) (*account.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	users, _, err := r.findUsers(ctx, tx, account.UserFilter{Email: &email})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, repository.ErrNotFound
	}

	user := users[0]

	if err := r.attachUserRecoveryCodes(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user recovery codes: %w", err)
	}

	if err := r.attachUserRoles(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user roles: %w", err)
	}
	for _, role := range user.Roles {
		if err := r.attachRolePermissions(ctx, tx, role); err != nil {
			return nil, fmt.Errorf("attach role permissions: %w", err)
		}
	}

	if err := r.attachUserGrants(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user grants: %w", err)
	}

	if err := r.attachUserDenials(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("attach user denials: %w", err)
	}

	return user, nil
}

func (r *AccountRepo) CountUsersByRoleID(ctx context.Context, roleID int) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, total, err := r.findUsers(ctx, tx, account.UserFilter{RoleID: &roleID})

	return total, err
}

func (r *AccountRepo) FindUsersPageBySearch(ctx context.Context, sortTopID int, search string, page, size int) ([]*account.User, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)

	return r.findUsers(ctx, tx, account.UserFilter{
		Search:    &search,
		SortTopID: sortTopID,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *AccountRepo) FindRoleByID(ctx context.Context, roleID int) (*account.Role, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	roles, _, err := r.findRoles(ctx, tx, account.RoleFilter{ID: &roleID})
	if len(roles) == 0 {
		return nil, repository.ErrNotFound
	}

	role := roles[0]

	if err := r.attachRolePermissions(ctx, tx, role); err != nil {
		return nil, fmt.Errorf("attach role permissions: %w", err)
	}

	return role, nil
}

func (r *AccountRepo) FindRoleByName(ctx context.Context, name string) (*account.Role, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	roles, _, err := r.findRoles(ctx, tx, account.RoleFilter{Name: &name})
	if len(roles) == 0 {
		return nil, repository.ErrNotFound
	}

	role := roles[0]

	if err := r.attachRolePermissions(ctx, tx, role); err != nil {
		return nil, fmt.Errorf("attach role permissions: %w", err)
	}

	return role, nil
}

func (r *AccountRepo) FindRolesByUserID(ctx context.Context, userID int) ([]*account.Role, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	roles, _, err := r.findRoles(ctx, tx, account.RoleFilter{UserID: &userID})

	return roles, err
}

func (r *AccountRepo) FindRoles(ctx context.Context, sortTopID int) ([]*account.Role, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findRoles(ctx, tx, account.RoleFilter{SortTopID: sortTopID})
}

func (r *AccountRepo) FindRolesPageBySearch(ctx context.Context, sortTopID int, search string, page, size int) ([]*account.Role, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)

	return r.findRoles(ctx, tx, account.RoleFilter{
		Search:    &search,
		SortTopID: sortTopID,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *AccountRepo) AddRole(ctx context.Context, role *account.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.createRole(ctx, tx, role); err != nil {
		return fmt.Errorf("create role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) SaveRole(ctx context.Context, role *account.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.updateRole(ctx, tx, role); err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) RemoveRole(ctx context.Context, roleID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.deleteRole(ctx, tx, roleID); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) FindRecoveryCodesByUserID(ctx context.Context, userID int) ([]account.RecoveryCode, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	recoveryCodes, _, err := r.findRecoveryCodes(ctx, tx, userID)

	return recoveryCodes, err
}

func (r *AccountRepo) AddUser(ctx context.Context, user *account.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.createUser(ctx, tx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) SaveUser(ctx context.Context, user *account.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.updateUser(ctx, tx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) findUsers(ctx context.Context, tx *Tx, filter account.UserFilter) ([]*account.User, int, error) {
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
		return nil, 0, err
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
			return nil, 0, err
		}

		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}

	return users, total, nil
}

type permissionFilter struct {
	roleID        *int
	grantsUserID  *int
	denialsUserID *int
}

func (r *AccountRepo) findPermissions(ctx context.Context, tx *Tx, filter permissionFilter) ([]string, int, error) {
	var joins []string
	var where []string
	var args []any

	if v := filter.roleID; v != nil {
		joins = append(joins, "INNER JOIN account__role_permissions AS rp ON p.id = rp.permission_id")
		where, args = append(where, "rp.role_id = ?"), append(args, *v)
	}
	if v := filter.grantsUserID; v != nil {
		joins = append(joins, "INNER JOIN account__user_grants AS ug ON p.id = ug.permission_id")
		where, args = append(where, "ug.user_id = ?"), append(args, *v)
	}
	if v := filter.denialsUserID; v != nil {
		joins = append(joins, "INNER JOIN account__user_denials AS ud ON p.id = ud.permission_id")
		where, args = append(where, "ud.user_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			name,
			COUNT(1) OVER () AS total
		FROM account__permissions AS p
		`+strings.Join(joins, "\n")+`
		`+whereSQL(where),
		args...,
	)
	if err != nil {
		return nil, 0, err
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
			return nil, 0, err
		}

		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}

	return permissions, total, nil
}

func (r *AccountRepo) upsertPermission(ctx context.Context, tx *Tx, name string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO account__permissions (
			name,
			created_at
		) VALUES (
			:name,
			:created_at
		)
		ON CONFLICT DO
			UPDATE SET
				name = :name
		RETURNING id
	`,
		sql.Named("name", name),
		sql.Named("created_at", Time(tx.now.UTC())),
	).Scan(&id)

	return id, err
}

func (r *AccountRepo) attachRolePermissions(ctx context.Context, tx *Tx, role *account.Role) error {
	permissions, _, err := r.findPermissions(ctx, tx, permissionFilter{roleID: &role.ID})
	if err != nil {
		return fmt.Errorf("find permissions: %w", err)
	}

	role.Permissions = permissions

	return nil
}

func (r *AccountRepo) findRoles(ctx context.Context, tx *Tx, filter account.RoleFilter) ([]*account.Role, int, error) {
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
		return nil, 0, err
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
			return nil, 0, err
		}

		roles = append(roles, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}

	return roles, total, nil
}

func (r *AccountRepo) createRole(ctx context.Context, tx *Tx, role *account.Role) error {
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
		if errors.Is(err, repository.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &repository.ConflictError{
				Map: errsx.Map{"name": errors.New("already in use")},
			})
		}

		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	role.ID = int(id)

	for _, name := range role.Permissions {
		permissionID, err := r.upsertPermission(ctx, tx, name)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
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
			return err
		}
	}

	return nil
}

func (r *AccountRepo) updateRole(ctx context.Context, tx *Tx, role *account.Role) error {
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
	if errors.Is(err, repository.ErrConflict) {
		return fmt.Errorf("%w: %w", err, &repository.ConflictError{
			Map: errsx.Map{"name": errors.New("already in use")},
		})
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__role_permissions WHERE role_id = :role_id",
		sql.Named("role_id", role.ID),
	)
	if err != nil {
		return err
	}

	for _, name := range role.Permissions {
		permissionID, err := r.upsertPermission(ctx, tx, name)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
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
			return err
		}
	}

	return nil
}

func (r *AccountRepo) deleteRole(ctx context.Context, tx *Tx, roleID int) error {
	_, err := tx.ExecContext(ctx,
		"DELETE FROM account__roles WHERE id = :id",
		sql.Named("id", roleID),
	)

	return err
}

func (r *AccountRepo) findRecoveryCodes(ctx context.Context, tx *Tx, userID int) ([]account.RecoveryCode, int, error) {
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
		return nil, 0, err
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
			return nil, 0, err
		}

		recoveryCodes = append(recoveryCodes, recoveryCode)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}

	return recoveryCodes, total, nil
}

func (r *AccountRepo) attachUserRecoveryCodes(ctx context.Context, tx *Tx, user *account.User) error {
	recoveryCodes, _, err := r.findRecoveryCodes(ctx, tx, user.ID)
	if err != nil {
		return fmt.Errorf("find recovery codes: %w", err)
	}

	if recoveryCodes != nil {
		user.RecoveryCodes = make([]string, len(recoveryCodes))

		for i, rc := range recoveryCodes {
			user.RecoveryCodes[i] = rc.String()
		}
	}

	return nil
}

func (r *AccountRepo) attachUserRoles(ctx context.Context, tx *Tx, user *account.User) error {
	roles, _, err := r.findRoles(ctx, tx, account.RoleFilter{UserID: &user.ID})
	if err != nil {
		return fmt.Errorf("find roles: %w", err)
	}

	user.Roles = roles

	return nil
}

func (r *AccountRepo) attachUserGrants(ctx context.Context, tx *Tx, user *account.User) error {
	grants, _, err := r.findPermissions(ctx, tx, permissionFilter{grantsUserID: &user.ID})
	if err != nil {
		return fmt.Errorf("find permissions: %w", err)
	}

	user.Grants = grants

	return nil
}

func (r *AccountRepo) attachUserDenials(ctx context.Context, tx *Tx, user *account.User) error {
	denials, _, err := r.findPermissions(ctx, tx, permissionFilter{denialsUserID: &user.ID})
	if err != nil {
		return fmt.Errorf("find permissions: %w", err)
	}

	user.Denials = denials

	return nil
}

func (r *AccountRepo) createUser(ctx context.Context, tx *Tx, user *account.User) error {
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
		if errors.Is(err, repository.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &repository.ConflictError{
				Map: errsx.Map{"email": errors.New("already in use")},
			})
		}

		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
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
			return err
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
			return err
		}
	}

	for _, grant := range user.Grants {
		permissionID, err := r.upsertPermission(ctx, tx, grant)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__user_grants (
				user_id,
				permission_id,
				created_at
			) VALUES (
				:user_id,
				:permission_id,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("permission_id", permissionID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	for _, denial := range user.Denials {
		permissionID, err := r.upsertPermission(ctx, tx, denial)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__user_denials (
				user_id,
				permission_id,
				created_at
			) VALUES (
				:user_id,
				:permission_id,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("permission_id", permissionID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *AccountRepo) updateUser(ctx context.Context, tx *Tx, user *account.User) error {
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
		if errors.Is(err, repository.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &repository.ConflictError{
				Map: errsx.Map{"email": errors.New("already in use")},
			})
		}

		return err
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__recovery_codes WHERE user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return err
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
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__user_roles WHERE user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return err
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
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__user_grants WHERE user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return err
	}

	for _, grant := range user.Grants {
		permissionID, err := r.upsertPermission(ctx, tx, grant)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__user_grants (
				user_id,
				permission_id,
				created_at
			) VALUES (
				:user_id,
				:permission_id,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("permission_id", permissionID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM account__user_denials WHERE user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return err
	}

	for _, denial := range user.Denials {
		permissionID, err := r.upsertPermission(ctx, tx, denial)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO account__user_denials (
				user_id,
				permission_id,
				created_at
			) VALUES (
				:user_id,
				:permission_id,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("permission_id", permissionID),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	return nil
}