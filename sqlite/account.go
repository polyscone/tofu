package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/i18n"
)

type AccountRepo struct {
	db *DB
}

func NewAccountRepo(ctx context.Context, db *DB, signInThrottleTTL time.Duration) (*AccountRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/account")
	if err != nil {
		return nil, fmt.Errorf("initialize account migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db, "account", migrations); err != nil {
		return nil, fmt.Errorf("migrate account: %w", err)
	}

	r := AccountRepo{db: db}

	// Background goroutine to clean up stale sign in attempt logs
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			validWindowStart := time.Now().Add(-signInThrottleTTL).UTC()
			if err := r.DeleteStaleSignInAttemptLogs(ctx, validWindowStart); err != nil {
				slog.Error("account repo: delete stale sign in attempt logs", "error", err)
			}
		}
	})

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
		return nil, app.ErrNotFound
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
		return nil, app.ErrNotFound
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

func (r *AccountRepo) CountUsers(ctx context.Context) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, total, err := r.findUsers(ctx, tx, account.UserFilter{})

	return total, err
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

func (r *AccountRepo) AddUser(ctx context.Context, user *account.User) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
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

func (r *AccountRepo) FindUsersPageBySearch(ctx context.Context, page, size, sortTopID int, sorts []string, search string) ([]*account.User, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)

	return r.findUsers(ctx, tx, account.UserFilter{
		Search:    &search,
		SortTopID: sortTopID,
		Sorts:     sorts,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *AccountRepo) SaveUser(ctx context.Context, user *account.User) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
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

func (r *AccountRepo) FindSignInAttemptLogByEmail(ctx context.Context, email string) (*account.SignInAttemptLog, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findSignInAttemptLog(ctx, tx, email)
}

func (r *AccountRepo) SaveSignInAttemptLog(ctx context.Context, log *account.SignInAttemptLog) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if log.Attempts == 0 {
		if err := r.deleteSignInAttemptLog(ctx, tx, log.Email); err != nil {
			return fmt.Errorf("delete sign in attempt log: %w", err)
		}
	} else {
		if err := r.upsertSignInAttemptLog(ctx, tx, log); err != nil {
			return fmt.Errorf("upsert sign in attempt log: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) CountStaleSignInAttemptLogs(ctx context.Context, validWindowStart time.Time) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	total, err := r.countStaleSignInAttemptLogs(ctx, tx, validWindowStart)

	return total, err
}

func (r *AccountRepo) DeleteStaleSignInAttemptLogs(ctx context.Context, validWindowStart time.Time) error {
	total, err := r.CountStaleSignInAttemptLogs(ctx, validWindowStart)
	if err != nil {
		return fmt.Errorf("count stale sign in attempt logs: %w", err)
	}
	if total == 0 {
		return nil
	}

	tx, err := r.db.BeginExclusiveTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.deleteStaleSignInAttemptLogs(ctx, tx, validWindowStart); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *AccountRepo) FindRoleByID(ctx context.Context, roleID int) (*account.Role, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	roles, _, err := r.findRoles(ctx, tx, account.RoleFilter{ID: &roleID})
	if err != nil {
		return nil, fmt.Errorf("find roles: %w", err)
	}
	if len(roles) == 0 {
		return nil, app.ErrNotFound
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
	if err != nil {
		return nil, fmt.Errorf("find roles: %w", err)
	}
	if len(roles) == 0 {
		return nil, app.ErrNotFound
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

func (r *AccountRepo) FindRolesPageBySearch(ctx context.Context, page, size, sortTopID int, sorts []string, search string) ([]*account.Role, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	limit, offset := pageLimitOffset(page, size)

	return r.findRoles(ctx, tx, account.RoleFilter{
		Search:    &search,
		SortTopID: sortTopID,
		Sorts:     sorts,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *AccountRepo) AddRole(ctx context.Context, role *account.Role) error {
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
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
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
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
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
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

func (r *AccountRepo) FindRecoveryCodesByUserID(ctx context.Context, userID int) ([][]byte, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	hashedCodes, _, err := r.findHashedRecoveryCodes(ctx, tx, userID)

	return hashedCodes, err
}

var findUserSortKeyCols = map[string]string{
	"email":        "u.email",
	"last-sign-in": "u.last_signed_in_at",
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
		where, args = append(where, "u.email like ?"), append(args, "%"+*v+"%")
	}
	if v := filter.RoleID; v != nil {
		joins = append(joins, "inner join account__user_roles as ur on u.id = ur.user_id")
		where, args = append(where, "ur.role_id = ?"), append(args, *v)
	}

	joins = append(joins, "left join account__totp_reset_requests as tr on u.id = tr.user_id")

	var sorts []string
	if filter.SortTopID != 0 {
		sorts, args = append(sorts, "case u.id when ? then 0 else 1 end asc"), append(args, filter.SortTopID)
	}

	if s := newSorts(filter.Sorts, findUserSortKeyCols); len(s) > 0 {
		sorts = append(sorts, s...)
	} else {
		sorts = append(sorts, "tr.requested_at desc, u.email asc")
	}

	rows, err := tx.QueryContext(ctx, `
		select
			u.id,
			u.email,
			u.hashed_password,
			u.totp_method,
			u.totp_tel,
			u.totp_key,
			u.totp_algorithm,
			u.totp_digits,
			u.totp_period,
			u.totp_verified_at,
			u.totp_activated_at,
			u.invited_at,
			u.signed_up_at,
			u.signed_up_system,
			u.signed_up_method,
			u.verified_at,
			u.activated_at,
			u.last_sign_in_attempt_at,
			u.last_sign_in_attempt_system,
			u.last_sign_in_attempt_method,
			u.last_signed_in_at,
			u.last_signed_in_system,
			u.last_signed_in_method,
			u.suspended_at,
			u.suspended_reason,
			tr.requested_at,
			tr.approved_at,
			count(*) over () as total
		from account__users as u
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
			(*Duration)(&user.TOTPPeriod),
			(*NullTime)(&user.TOTPVerifiedAt),
			(*NullTime)(&user.TOTPActivatedAt),
			(*NullTime)(&user.InvitedAt),
			(*NullTime)(&user.SignedUpAt),
			&user.SignedUpSystem,
			&user.SignedUpMethod,
			(*NullTime)(&user.VerifiedAt),
			(*NullTime)(&user.ActivatedAt),
			(*NullTime)(&user.LastSignInAttemptAt),
			&user.LastSignInAttemptSystem,
			&user.LastSignInAttemptMethod,
			(*NullTime)(&user.LastSignedInAt),
			&user.LastSignedInSystem,
			&user.LastSignedInMethod,
			(*NullTime)(&user.SuspendedAt),
			&user.SuspendedReason,
			(*NullTime)(&user.TOTPResetRequestedAt),
			(*NullTime)(&user.TOTPResetApprovedAt),
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

func (r *AccountRepo) attachUserRecoveryCodes(ctx context.Context, tx *Tx, user *account.User) error {
	hashedCodes, _, err := r.findHashedRecoveryCodes(ctx, tx, user.ID)
	if err != nil {
		return fmt.Errorf("find hashed recovery codes: %w", err)
	}

	if hashedCodes != nil {
		user.HashedRecoveryCodes = make([][]byte, len(hashedCodes))

		copy(user.HashedRecoveryCodes, hashedCodes)
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
		insert into account__users (
			email,
			hashed_password,
			totp_method,
			totp_tel,
			totp_key,
			totp_algorithm,
			totp_digits,
			totp_period,
			totp_verified_at,
			totp_activated_at,
			invited_at,
			signed_up_at,
			signed_up_system,
			signed_up_method,
			verified_at,
			activated_at,
			last_sign_in_attempt_at,
			last_sign_in_attempt_system,
			last_sign_in_attempt_method,
			last_signed_in_at,
			last_signed_in_system,
			last_signed_in_method,
			suspended_at,
			suspended_reason,
			created_at
		) values (
			:email,
			:hashed_password,
			:totp_method,
			:totp_tel,
			:totp_key,
			:totp_algorithm,
			:totp_digits,
			:totp_period,
			:totp_verified_at,
			:totp_activated_at,
			:invited_at,
			:signed_up_at,
			:signed_up_system,
			:signed_up_method,
			:verified_at,
			:activated_at,
			:last_sign_in_attempt_at,
			:last_sign_in_attempt_system,
			:last_sign_in_attempt_method,
			:last_signed_in_at,
			:last_signed_in_system,
			:last_signed_in_method,
			:suspended_at,
			:suspended_reason,
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
		sql.Named("totp_period", Duration(user.TOTPPeriod)),
		sql.Named("totp_verified_at", NullTime(user.TOTPVerifiedAt)),
		sql.Named("totp_activated_at", NullTime(user.TOTPActivatedAt)),
		sql.Named("invited_at", NullTime(user.InvitedAt)),
		sql.Named("signed_up_at", NullTime(user.SignedUpAt)),
		sql.Named("signed_up_system", user.SignedUpSystem),
		sql.Named("signed_up_method", user.SignedUpMethod),
		sql.Named("verified_at", NullTime(user.VerifiedAt)),
		sql.Named("activated_at", NullTime(user.ActivatedAt)),
		sql.Named("last_sign_in_attempt_at", NullTime(user.LastSignInAttemptAt)),
		sql.Named("last_sign_in_attempt_system", user.LastSignInAttemptSystem),
		sql.Named("last_sign_in_attempt_method", user.LastSignInAttemptMethod),
		sql.Named("last_signed_in_at", NullTime(user.LastSignedInAt)),
		sql.Named("last_signed_in_system", user.LastSignedInSystem),
		sql.Named("last_signed_in_method", user.LastSignedInMethod),
		sql.Named("suspended_at", NullTime(user.SuspendedAt)),
		sql.Named("suspended_reason", user.SuspendedReason),
		sql.Named("created_at", Time(tx.now.UTC())),
	)
	if err != nil {
		if errors.Is(err, app.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &app.ConflictError{
				Map: errsx.Map{"email": i18n.M("user.email:repo.error.conflict", "value", user.Email)},
			})
		}

		return err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	user.ID = int(lastID)

	if !user.TOTPResetRequestedAt.IsZero() || !user.TOTPResetApprovedAt.IsZero() {
		_, err := tx.ExecContext(ctx, `
			insert into account__totp_reset_requests (
				user_id,
				requested_at,
				approved_at,
				created_at
			) values (
				:user_id,
				:requested_at,
				:approved_at,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("requested_at", NullTime(user.TOTPResetRequestedAt)),
			sql.Named("approved_at", NullTime(user.TOTPResetApprovedAt)),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	for _, rc := range user.HashedRecoveryCodes {
		_, err := tx.ExecContext(ctx, `
			insert into account__recovery_codes (
				user_id,
				hashed_code,
				created_at
			) values (
				:user_id,
				:hashed_code,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("hashed_code", rc),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	for _, role := range user.Roles {
		_, err := tx.ExecContext(ctx, `
			insert into account__user_roles (
				user_id,
				role_id,
				created_at
			) values (
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
			insert into account__user_grants (
				user_id,
				permission_id,
				created_at
			) values (
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
			insert into account__user_denials (
				user_id,
				permission_id,
				created_at
			) values (
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
		update account__users set
			email = :email,
			hashed_password = :hashed_password,
			totp_method = :totp_method,
			totp_tel = :totp_tel,
			totp_key = :totp_key,
			totp_algorithm = :totp_algorithm,
			totp_digits = :totp_digits,
			totp_period = :totp_period,
			totp_verified_at = :totp_verified_at,
			totp_activated_at = :totp_activated_at,
			invited_at = :invited_at,
			signed_up_at = :signed_up_at,
			signed_up_system = :signed_up_system,
			signed_up_method = :signed_up_method,
			verified_at = :verified_at,
			activated_at = :activated_at,
			last_sign_in_attempt_at = :last_sign_in_attempt_at,
			last_sign_in_attempt_system = :last_sign_in_attempt_system,
			last_sign_in_attempt_method = :last_sign_in_attempt_method,
			last_signed_in_at = :last_signed_in_at,
			last_signed_in_system = :last_signed_in_system,
			last_signed_in_method = :last_signed_in_method,
			suspended_at = :suspended_at,
			suspended_reason = :suspended_reason,
			updated_at = :updated_at
		where id = :id
	`,
		sql.Named("id", user.ID),
		sql.Named("email", user.Email),
		sql.Named("hashed_password", user.HashedPassword),
		sql.Named("totp_method", user.TOTPMethod),
		sql.Named("totp_tel", user.TOTPTel),
		sql.Named("totp_key", user.TOTPKey),
		sql.Named("totp_algorithm", user.TOTPAlgorithm),
		sql.Named("totp_digits", user.TOTPDigits),
		sql.Named("totp_period", Duration(user.TOTPPeriod)),
		sql.Named("totp_verified_at", NullTime(user.TOTPVerifiedAt)),
		sql.Named("totp_activated_at", NullTime(user.TOTPActivatedAt)),
		sql.Named("invited_at", NullTime(user.InvitedAt)),
		sql.Named("signed_up_at", NullTime(user.SignedUpAt)),
		sql.Named("signed_up_system", user.SignedUpSystem),
		sql.Named("signed_up_method", user.SignedUpMethod),
		sql.Named("verified_at", NullTime(user.VerifiedAt)),
		sql.Named("activated_at", NullTime(user.ActivatedAt)),
		sql.Named("last_sign_in_attempt_at", NullTime(user.LastSignInAttemptAt)),
		sql.Named("last_sign_in_attempt_system", user.LastSignInAttemptSystem),
		sql.Named("last_sign_in_attempt_method", user.LastSignInAttemptMethod),
		sql.Named("last_signed_in_at", NullTime(user.LastSignedInAt)),
		sql.Named("last_signed_in_system", user.LastSignedInSystem),
		sql.Named("last_signed_in_method", user.LastSignedInMethod),
		sql.Named("suspended_at", NullTime(user.SuspendedAt)),
		sql.Named("suspended_reason", user.SuspendedReason),
		sql.Named("updated_at", NullTime(tx.now.UTC())),
	)
	if err != nil {
		if errors.Is(err, app.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &app.ConflictError{
				Map: errsx.Map{"email": i18n.M("user.email:repo.error.conflict", "value", user.Email)},
			})
		}

		return err
	}

	if user.TOTPResetRequestedAt.IsZero() && user.TOTPResetApprovedAt.IsZero() {
		_, err := tx.ExecContext(ctx,
			"delete from account__totp_reset_requests where user_id = :user_id",
			sql.Named("user_id", user.ID),
		)
		if err != nil {
			return err
		}
	} else {
		_, err := tx.ExecContext(ctx, `
			insert into account__totp_reset_requests (
				user_id,
				requested_at,
				approved_at,
				created_at
			) values (
				:user_id,
				:requested_at,
				:approved_at,
				:created_at
			)
			on conflict (user_id) do
				update set
					requested_at = excluded.requested_at,
					approved_at = excluded.approved_at,
					updated_at = :updated_at
				where
					ifnull(requested_at, '') != ifnull(excluded.requested_at, '') or
					ifnull(approved_at, '') != ifnull(excluded.approved_at, '')
		`,
			sql.Named("user_id", user.ID),
			sql.Named("requested_at", NullTime(user.TOTPResetRequestedAt)),
			sql.Named("approved_at", NullTime(user.TOTPResetApprovedAt)),
			sql.Named("created_at", Time(tx.now.UTC())),
			sql.Named("updated_at", NullTime(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		"delete from account__recovery_codes where user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return err
	}

	for _, rc := range user.HashedRecoveryCodes {
		_, err := tx.ExecContext(ctx, `
			insert into account__recovery_codes (
				user_id,
				hashed_code,
				created_at
			) values (
				:user_id,
				:hashed_code,
				:created_at
			)
		`,
			sql.Named("user_id", user.ID),
			sql.Named("hashed_code", rc),
			sql.Named("created_at", Time(tx.now.UTC())),
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		"delete from account__user_roles where user_id = :user_id",
		sql.Named("user_id", user.ID),
	)
	if err != nil {
		return err
	}

	for _, role := range user.Roles {
		_, err := tx.ExecContext(ctx, `
			insert into account__user_roles (
				user_id,
				role_id,
				created_at
			) values (
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
		"delete from account__user_grants where user_id = :user_id",
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
			insert into account__user_grants (
				user_id,
				permission_id,
				created_at
			) values (
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
		"delete from account__user_denials where user_id = :user_id",
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
			insert into account__user_denials (
				user_id,
				permission_id,
				created_at
			) values (
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

func (r *AccountRepo) findSignInAttemptLog(ctx context.Context, tx *Tx, email string) (*account.SignInAttemptLog, error) {
	log := account.SignInAttemptLog{Email: email}

	err := tx.QueryRowContext(ctx, `
		select
			attempts,
			last_attempt_at
		from account__sign_in_attempt_logs
		where email = :email
	`,
		sql.Named("email", email),
	).Scan(
		&log.Attempts,
		(*Time)(&log.LastAttemptAt),
	)
	if errors.Is(err, app.ErrNotFound) {
		err = nil
	}

	return &log, err
}

func (r *AccountRepo) upsertSignInAttemptLog(ctx context.Context, tx *Tx, log *account.SignInAttemptLog) error {
	_, err := tx.ExecContext(ctx, `
		insert into account__sign_in_attempt_logs (
			email,
			attempts,
			last_attempt_at,
			created_at
		) values (
			:email,
			:attempts,
			:last_attempt_at,
			:created_at
		)
		on conflict (email) do
			update set
				attempts = excluded.attempts,
				last_attempt_at = excluded.last_attempt_at,
				updated_at = :updated_at
			where
				attempts != excluded.attempts or
				last_attempt_at != excluded.last_attempt_at
	`,
		sql.Named("email", log.Email),
		sql.Named("attempts", log.Attempts),
		sql.Named("last_attempt_at", Time(log.LastAttemptAt.UTC())),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", NullTime(tx.now.UTC())),
	)

	return err
}

func (r *AccountRepo) deleteSignInAttemptLog(ctx context.Context, tx *Tx, email string) error {
	_, err := tx.ExecContext(ctx, `
		delete from account__sign_in_attempt_logs
		where email = :email
	`,
		sql.Named("email", email),
	)

	return err
}

func (r *AccountRepo) countStaleSignInAttemptLogs(ctx context.Context, tx *Tx, validWindowStart time.Time) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx, `
		select count(*) from account__sign_in_attempt_logs
		where last_attempt_at <= :valid_window_start
	`,
		sql.Named("valid_window_start", Time(validWindowStart)),
	).Scan(&count)

	return count, err
}

func (r *AccountRepo) deleteStaleSignInAttemptLogs(ctx context.Context, tx *Tx, validWindowStart time.Time) error {
	_, err := tx.ExecContext(ctx, `
		delete from account__sign_in_attempt_logs
		where last_attempt_at <= :valid_window_start
	`,
		sql.Named("valid_window_start", Time(validWindowStart)),
	)

	return err
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
		joins = append(joins, "inner join account__role_permissions as rp on p.id = rp.permission_id")
		where, args = append(where, "rp.role_id = ?"), append(args, *v)
	}
	if v := filter.grantsUserID; v != nil {
		joins = append(joins, "inner join account__user_grants as ug on p.id = ug.permission_id")
		where, args = append(where, "ug.user_id = ?"), append(args, *v)
	}
	if v := filter.denialsUserID; v != nil {
		joins = append(joins, "inner join account__user_denials as ud on p.id = ud.permission_id")
		where, args = append(where, "ud.user_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		select
			name,
			count(*) over () as total
		from account__permissions as p
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
		insert into account__permissions (
			name,
			created_at
		) values (
			:name,
			:created_at
		)
		on conflict (name) do
			update set
				name = excluded.name,
				updated_at = :updated_at
		returning id
	`,
		sql.Named("name", name),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", NullTime(tx.now.UTC())),
	).Scan(&id)

	return id, err
}

var findRolesSortKeyCols = map[string]string{
	"name": "r.name",
}

func (r *AccountRepo) findRoles(ctx context.Context, tx *Tx, filter account.RoleFilter) ([]*account.Role, int, error) {
	var joins []string
	var where []string
	var args []any

	if v := filter.ID; v != nil {
		where, args = append(where, "r.id = ?"), append(args, *v)
	}
	if v := filter.UserID; v != nil {
		joins = append(joins, "inner join account__user_roles as ur on r.id = ur.role_id")
		where, args = append(where, "ur.user_id = ?"), append(args, *v)
	}
	if v := filter.Name; v != nil && *v != "" {
		where, args = append(where, "r.name = ?"), append(args, *v)
	}
	if v := filter.Search; v != nil && *v != "" {
		where, args = append(where, "r.name like ?"), append(args, "%"+*v+"%")
	}

	var sorts []string
	if filter.SortTopID != 0 {
		sorts, args = append(sorts, "case r.id when ? then 0 else 1 end asc"), append(args, filter.SortTopID)
	}

	if s := newSorts(filter.Sorts, findRolesSortKeyCols); len(s) > 0 {
		sorts = append(sorts, s...)
	} else {
		sorts = append(sorts, "r.name asc")
	}

	rows, err := tx.QueryContext(ctx, `
		select
			id,
			name,
			description,
			count(*) over () as total
		from account__roles as r
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

func (r *AccountRepo) attachRolePermissions(ctx context.Context, tx *Tx, role *account.Role) error {
	permissions, _, err := r.findPermissions(ctx, tx, permissionFilter{roleID: &role.ID})
	if err != nil {
		return fmt.Errorf("find permissions: %w", err)
	}

	role.Permissions = permissions

	return nil
}

func (r *AccountRepo) createRole(ctx context.Context, tx *Tx, role *account.Role) error {
	res, err := tx.ExecContext(ctx, `
		insert into account__roles (
			name,
			description,
			created_at
		) values (
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
		if errors.Is(err, app.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &app.ConflictError{
				Map: errsx.Map{"name": i18n.M("role.name:repo.error.conflict", "value", role.Name)},
			})
		}

		return err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	role.ID = int(lastID)

	for _, name := range role.Permissions {
		permissionID, err := r.upsertPermission(ctx, tx, name)
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			insert into account__role_permissions (
				role_id,
				permission_id,
				created_at
			) values (
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
		update account__roles set
			name = :name,
			description = :description,
			updated_at = :updated_at
		where id = :id
	`,
		sql.Named("id", role.ID),
		sql.Named("name", role.Name),
		sql.Named("description", role.Description),
		sql.Named("updated_at", NullTime(tx.now.UTC())),
	)
	if err != nil {
		if errors.Is(err, app.ErrConflict) {
			return fmt.Errorf("%w: %w", err, &app.ConflictError{
				Map: errsx.Map{"name": i18n.M("role.name:repo.error.conflict", "value", role.Name)},
			})
		}

		return err
	}

	_, err = tx.ExecContext(ctx,
		"delete from account__role_permissions where role_id = :role_id",
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
			insert into account__role_permissions (
				role_id,
				permission_id,
				created_at
			) values (
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
		"delete from account__roles where id = :id",
		sql.Named("id", roleID),
	)

	return err
}

func (r *AccountRepo) findHashedRecoveryCodes(ctx context.Context, tx *Tx, userID int) ([][]byte, int, error) {
	rows, err := tx.QueryContext(ctx, `
		select
			hashed_code,
			count(*) over () as total
		from account__recovery_codes
		where user_id = :user_id
	`,
		sql.Named("user_id", userID),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	var hashedCodes [][]byte
	for rows.Next() {
		var hashedCode []byte

		err := rows.Scan(
			&hashedCode,
			&total,
		)
		if err != nil {
			return nil, 0, err
		}

		hashedCodes = append(hashedCodes, hashedCode)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}

	return hashedCodes, total, nil
}
