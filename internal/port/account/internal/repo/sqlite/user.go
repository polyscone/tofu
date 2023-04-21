package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

//go:embed "migrations"
var migrations embed.FS

type UserRepo struct {
	db     *sqlite.DB
	secret []byte
}

func NewUserRepo(ctx context.Context, db *sqlite.DB, secret []byte) (*UserRepo, error) {
	if err := db.MigrateFS(ctx, "account", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	repo := UserRepo{
		db:     db,
		secret: secret,
	}

	return &repo, nil
}

func (r *UserRepo) findBy(ctx context.Context, where string, args sqlite.Args) (domain.User, error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return domain.User{}, errors.Tracef(err)
	}
	defer tx.Rollback()

	var id uuid.V4
	var email text.Email
	var hashedPassword []byte
	var totpKey []byte
	var totpVerifiedAt sql.NullTime
	var activatedAt sql.NullTime

	stmt := fmt.Sprintf(`
		SELECT u.id, u.email, u.hashed_password, u.totp_key, u.totp_verified_at, u.activated_at
		FROM account__users AS u
		WHERE %v;
	`, where)
	err = tx.QueryRow(ctx, stmt, args).Scan(&id, &email, &hashedPassword, &totpKey, &totpVerifiedAt, &activatedAt)
	if err != nil {
		return domain.User{}, errors.Tracef(err)
	}

	roles, err := r.findRolesByUserID(ctx, tx, id)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return domain.User{}, errors.Tracef(err)
	}

	res := domain.NewUser(id)

	res.Email = email
	res.HashedPassword = hashedPassword
	res.TOTPKey = totpKey
	res.TOTPVerifiedAt = totpVerifiedAt.Time
	res.Roles = roles
	res.ActivatedAt = activatedAt.Time

	return res, errors.Tracef(tx.Commit())
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.V4) (domain.User, error) {
	return r.findBy(ctx, `u.id = :id`, sqlite.Args{"id": id})
}

func (r *UserRepo) FindByEmail(ctx context.Context, email text.Email) (domain.User, error) {
	return r.findBy(ctx, `u.email = :email`, sqlite.Args{"email": email})
}

func (r *UserRepo) Add(ctx context.Context, u domain.User) error {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	defer tx.Rollback()

	if u.HashedPassword == nil {
		u.HashedPassword = make([]byte, 0)
	}

	if u.TOTPKey == nil {
		u.TOTPKey = make([]byte, 0)
	}

	stmt, args := `
		INSERT INTO account__users
			(id, email, hashed_password, totp_key)
		VALUES
			(:id, :email, :hashed_password, :totp_key);
	`, sqlite.Args{
		"id":              u.ID,
		"email":           u.Email,
		"hashed_password": u.HashedPassword,
		"totp_key":        u.TOTPKey,
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return errors.Tracef(err)
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

func (r *UserRepo) Save(ctx context.Context, u domain.User) error {
	stmt, args := `
		UPDATE account__users SET
			email = :email,
			hashed_password = :hashed_password,
			totp_key = :totp_key,
			totp_verified_at = :totp_verified_at,
			activated_at = :activated_at
		WHERE id = :id;
	`, sqlite.Args{
		"id":               u.ID,
		"email":            u.Email,
		"hashed_password":  u.HashedPassword,
		"totp_key":         u.TOTPKey,
		"totp_verified_at": sqlite.NewNullTime(u.TOTPVerifiedAt.UTC()),
		"activated_at":     sqlite.NewNullTime(u.ActivatedAt.UTC()),
	}
	_, err := r.db.Exec(ctx, stmt, args)

	return errors.Tracef(err)
}
